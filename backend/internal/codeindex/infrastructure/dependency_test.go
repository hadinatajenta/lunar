package infrastructure

import (
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

func TestExtractServiceFromURL(t *testing.T) {
	testCases := []struct {
		rawURL      string
		selfService string
		expected    string
	}{
		{"http://billing-service:8080/api/invoices", "payment", "billing-service"},
		{"https://order-service/internal/orders", "payment", "order-service"},
		{"http://localhost:3000/api", "payment", ""},
		{"http://127.0.0.1:8080/api", "payment", ""},
		{"http://10.0.0.5:8080/api", "payment", ""},
		{"http://billing-service/api", "billing", ""},
		{"http://${BILLING_HOST}/api", "payment", ""},
		{"http://api-env.internal/api", "payment", ""},
		{"", "payment", ""},
	}

	for _, testCase := range testCases {
		extracted := extractServiceFromURL(testCase.rawURL, testCase.selfService)
		if extracted != testCase.expected {
			t.Errorf("extractServiceFromURL(%q, %q) = %q, want %q", testCase.rawURL, testCase.selfService, extracted, testCase.expected)
		}
	}
}

func TestHTTPClientDependency(t *testing.T) {
	dependency, isFound := httpClientDependency("payment-service", "internal/client.go", "http://billing-service:9000/api/invoices")
	if !isFound {
		t.Fatal("expected a dependency for the billing-service endpoint")
	}
	if dependency.FromService != "payment-service" {
		t.Errorf("unexpected from_service %q", dependency.FromService)
	}
	if dependency.ToService != "billing-service" {
		t.Errorf("unexpected to_service %q", dependency.ToService)
	}
	if dependency.CallType != string(codeindexdomain.CallTypeHTTPClient) {
		t.Errorf("unexpected call_type %q", dependency.CallType)
	}
	if dependency.Endpoint != "http://billing-service:9000/api/invoices" {
		t.Errorf("unexpected endpoint %q", dependency.Endpoint)
	}
	if dependency.SourceFile != "internal/client.go" {
		t.Errorf("unexpected source file %q", dependency.SourceFile)
	}

	if _, isFound := httpClientDependency("payment-service", "internal/client.go", "http://localhost:9000/api"); isFound {
		t.Error("expected no dependency for localhost endpoints")
	}
}

func TestKafkaDependency(t *testing.T) {
	publishDependency := kafkaDependency("payment-service", "internal/producer.go", "payment-events", codeindexdomain.CallTypeKafkaPublish)
	if publishDependency.CallType != string(codeindexdomain.CallTypeKafkaPublish) {
		t.Errorf("unexpected call_type %q", publishDependency.CallType)
	}
	if publishDependency.Topic != "payment-events" {
		t.Errorf("unexpected topic %q", publishDependency.Topic)
	}
	if publishDependency.ToService != "" {
		t.Errorf("expected topic-only edge without to_service, got %q", publishDependency.ToService)
	}

	subscribeDependency := kafkaDependency("payment-service", "internal/consumer.go", "order-created", codeindexdomain.CallTypeKafkaSubscribe)
	if subscribeDependency.CallType != string(codeindexdomain.CallTypeKafkaSubscribe) {
		t.Errorf("unexpected call_type %q", subscribeDependency.CallType)
	}
	if subscribeDependency.Topic != "order-created" {
		t.Errorf("unexpected topic %q", subscribeDependency.Topic)
	}
}

func TestIsIPAddress(t *testing.T) {
	for _, value := range []string{"10.0.0.5", "192.168.1.1", "localhost", "127.0.0.1"} {
		if !isIPAddress(value) {
			t.Errorf("expected %q to be treated as non-routable host", value)
		}
	}
	if isIPAddress("billing-service") {
		t.Errorf("expected hostname to be treated as a service")
	}
}
