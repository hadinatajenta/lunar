package infrastructure

import (
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const nodeFixture = `const express = require('express');
const router = express.Router();

router.get('/api/merchants', getMerchants);
router.post('/api/merchants', createMerchant);

async function loadMerchant() {
  const response = await axios.get('http://merchant-service/api/v1/merchants');
  return response.data;
}
`

func TestExtractNodeReportsRouteAndFunctionRanges(t *testing.T) {
	sourcePath, content := writeFixtureFile(t, "merchant.js", nodeFixture)
	chunks, _ := extractNode(content, "routes/merchant.js", "mms-api")
	assertChunkRangesMatchSource(t, sourcePath, chunks)

	routeChunk := findChunkContaining(chunks, "router.get('/api/merchants'")
	if routeChunk == nil {
		t.Fatal("expected a route chunk for /api/merchants")
	}
	if routeChunk.ChunkType != string(codeindexdomain.ChunkTypeRoute) {
		t.Errorf("expected route chunk type, got %q", routeChunk.ChunkType)
	}
	if routeChunk.StartLine != 4 || routeChunk.EndLine != 4 {
		t.Errorf("expected route on line 4, got %d-%d", routeChunk.StartLine, routeChunk.EndLine)
	}

	functionChunk := findChunkContaining(chunks, "async function loadMerchant")
	if functionChunk == nil {
		t.Fatal("expected a function chunk for loadMerchant")
	}
	if functionChunk.StartLine != 7 || functionChunk.EndLine != 10 {
		t.Errorf("expected loadMerchant on lines 7-10, got %d-%d", functionChunk.StartLine, functionChunk.EndLine)
	}
}

func TestExtractNodeExtractsHTTPClientDependency(t *testing.T) {
	_, content := writeFixtureFile(t, "merchant.js", nodeFixture)
	_, dependencies := extractNode(content, "routes/merchant.js", "mms-api")

	for _, dependency := range dependencies {
		if dependency.CallType != string(codeindexdomain.CallTypeHTTPClient) {
			continue
		}
		if dependency.ToService != "merchant-service" {
			t.Errorf("expected to_service merchant-service, got %q", dependency.ToService)
		}
		if dependency.Endpoint != "http://merchant-service/api/v1/merchants" {
			t.Errorf("unexpected endpoint %q", dependency.Endpoint)
		}
		return
	}
	t.Fatal("expected an http_client dependency for merchant-service")
}
