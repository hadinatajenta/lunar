package infrastructure

import (
	"strings"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const goFixtureTemplate = `package handler

import "net/http"

func RegisterRoutes(router *gin.Engine) {
	router.GET("/api/payments", getPayments)
	mux.HandleFunc("/api/logbooks", getLogbooks)
}

func publishPayment(writer *kafka.Writer) error {
	return writer.WriteMessages(ctx, kafka.Message{Topic: "payment-events"})
}

func subscribeOrder(reader *kafka.Reader) {
	reader.Subscribe("order-created")
	http.Get("http://billing-service/api/invoices")
}

func handleRequest(request *Request) {
	request.AppJenis = 2
	payload["app_jenis"] = 2
}

@COMMENT@ Status aktif pengguna
const (
	StatusActive = iota
	StatusInactive
)

const StatusArchived = "archived"
`

func goFixtureSource() string {
	return strings.Replace(goFixtureTemplate, "@COMMENT@", lineCommentPrefix, 1)
}

func extractGoFixture(t *testing.T) (string, []codeindexdomain.CodeChunk, []codeindexdomain.ServiceDependency) {
	t.Helper()

	sourcePath, content := writeFixtureFile(t, "routes.go", goFixtureSource())
	chunks, dependencies := extractGo(content, "handler/routes.go", "micro-payment")
	assertChunkRangesMatchSource(t, sourcePath, chunks)
	return sourcePath, chunks, dependencies
}

func TestExtractGoReportsRouteLineRange(t *testing.T) {
	_, chunks, _ := extractGoFixture(t)

	routeChunk := findChunkContaining(chunks, `router.GET("/api/payments"`)
	if routeChunk == nil {
		t.Fatal("expected a route chunk for /api/payments")
	}
	if routeChunk.ChunkType != string(codeindexdomain.ChunkTypeRoute) {
		t.Errorf("expected route chunk type, got %q", routeChunk.ChunkType)
	}
	if routeChunk.StartLine != 6 || routeChunk.EndLine != 6 {
		t.Errorf("expected route on lines 6-6, got %d-%d", routeChunk.StartLine, routeChunk.EndLine)
	}
	if routeChunk.RawContent != "\trouter.GET(\"/api/payments\", getPayments)" {
		t.Errorf("unexpected route chunk content %q", routeChunk.RawContent)
	}

	muxChunk := findChunkContaining(chunks, `HandleFunc("/api/logbooks"`)
	if muxChunk == nil || muxChunk.StartLine != 7 || muxChunk.EndLine != 7 {
		t.Errorf("expected mux route on lines 7-7, got %+v", muxChunk)
	}
}

func TestExtractGoReportsFunctionBodyRange(t *testing.T) {
	_, chunks, _ := extractGoFixture(t)

	functionChunk := findChunkContaining(chunks, "func publishPayment")
	if functionChunk == nil {
		t.Fatal("expected a function chunk for publishPayment")
	}
	if functionChunk.ChunkType != string(codeindexdomain.ChunkTypeCodeBlock) {
		t.Errorf("expected code_block chunk type, got %q", functionChunk.ChunkType)
	}
	if functionChunk.StartLine != 10 || functionChunk.EndLine != 12 {
		t.Errorf("expected publishPayment on lines 10-12, got %d-%d", functionChunk.StartLine, functionChunk.EndLine)
	}
}

func TestExtractGoExtractsKafkaDependencies(t *testing.T) {
	_, _, dependencies := extractGoFixture(t)

	var publishTopic, subscribeTopic string
	for _, dependency := range dependencies {
		switch dependency.CallType {
		case string(codeindexdomain.CallTypeKafkaPublish):
			publishTopic = dependency.Topic
		case string(codeindexdomain.CallTypeKafkaSubscribe):
			subscribeTopic = dependency.Topic
		}
	}

	if publishTopic != "payment-events" {
		t.Errorf("expected publish topic payment-events, got %q", publishTopic)
	}
	if subscribeTopic != "order-created" {
		t.Errorf("expected subscribe topic order-created, got %q", subscribeTopic)
	}
}

func TestExtractGoExtractsHTTPClientDependency(t *testing.T) {
	_, _, dependencies := extractGoFixture(t)

	for _, dependency := range dependencies {
		if dependency.CallType != string(codeindexdomain.CallTypeHTTPClient) {
			continue
		}
		if dependency.ToService != "billing-service" {
			t.Errorf("expected to_service billing-service, got %q", dependency.ToService)
		}
		if dependency.Endpoint != "http://billing-service/api/invoices" {
			t.Errorf("unexpected endpoint %q", dependency.Endpoint)
		}
		if dependency.SourceFile != "handler/routes.go" {
			t.Errorf("unexpected source file %q", dependency.SourceFile)
		}
		return
	}
	t.Fatal("expected an http_client dependency for billing-service")
}

func TestExtractGoExtractsEnumAndFieldChunks(t *testing.T) {
	_, chunks, _ := extractGoFixture(t)

	enumChunk := findChunkContaining(chunks, "StatusActive")
	if enumChunk == nil {
		t.Fatal("expected an enum chunk containing StatusActive")
	}
	if enumChunk.ChunkType != string(codeindexdomain.ChunkTypeEnumDef) {
		t.Errorf("expected enum_definition chunk type, got %q", enumChunk.ChunkType)
	}
	if enumChunk.StartLine != 24 || enumChunk.EndLine != 28 {
		t.Errorf("expected const block with leading comment on lines 24-28, got %d-%d", enumChunk.StartLine, enumChunk.EndLine)
	}
	if !strings.HasPrefix(enumChunk.RawContent, lineCommentPrefix+" Status aktif pengguna") {
		t.Errorf("expected leading source comment line in chunk, got %q", enumChunk.RawContent)
	}
	if count := countChunksOfType(chunks, string(codeindexdomain.ChunkTypeEnumDef)); count != 2 {
		t.Errorf("expected 2 enum chunks (const block and single const), got %d", count)
	}

	fieldChunk := findChunkWithTrimmedContent(chunks, "request.AppJenis = 2")
	if fieldChunk == nil {
		t.Fatal("expected a field assignment chunk for request.AppJenis")
	}
	if fieldChunk.StartLine != 20 || fieldChunk.EndLine != 20 {
		t.Errorf("expected field assignment on line 20, got %d-%d", fieldChunk.StartLine, fieldChunk.EndLine)
	}

	payloadChunk := findChunkWithTrimmedContent(chunks, `payload["app_jenis"] = 2`)
	if payloadChunk == nil || payloadChunk.StartLine != 21 {
		t.Errorf("expected map assignment on line 21, got %+v", payloadChunk)
	}
}
