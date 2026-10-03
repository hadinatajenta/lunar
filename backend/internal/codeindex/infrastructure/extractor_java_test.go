package infrastructure

import (
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const javaFixture = `package com.example.aurora;

@RestController
@RequestMapping("/api/aurora")
public class AuroraController {

    @GetMapping("/transactions")
    public List<Transaction> getTransactions() {
        return service.findAll();
    }

    @PostMapping("/transactions")
    public Transaction createTransaction(TransactionRequest request) {
        return service.create(request);
    }
}
`

func TestExtractJavaReportsRouteRanges(t *testing.T) {
	sourcePath, content := writeFixtureFile(t, "AuroraController.java", javaFixture)
	chunks, _ := extractJava(content, "AuroraController.java", "aurora")
	assertChunkRangesMatchSource(t, sourcePath, chunks)

	if count := countChunksOfType(chunks, string(codeindexdomain.ChunkTypeRoute)); count != 3 {
		t.Fatalf("expected 3 route chunks, got %d", count)
	}

	getChunk := findChunkContaining(chunks, `@GetMapping("/transactions")`)
	if getChunk == nil {
		t.Fatal("expected a route chunk for @GetMapping")
	}
	if getChunk.StartLine != 7 || getChunk.EndLine != 7 {
		t.Errorf("expected @GetMapping on line 7, got %d-%d", getChunk.StartLine, getChunk.EndLine)
	}

	classChunk := findChunkContaining(chunks, `@RequestMapping("/api/aurora")`)
	if classChunk == nil || classChunk.StartLine != 4 {
		t.Errorf("expected @RequestMapping on line 4, got %+v", classChunk)
	}
}

func TestExtractJavaReportsMethodRanges(t *testing.T) {
	sourcePath, content := writeFixtureFile(t, "AuroraController.java", javaFixture)
	chunks, _ := extractJava(content, "AuroraController.java", "aurora")
	assertChunkRangesMatchSource(t, sourcePath, chunks)

	getMethod := findChunkContaining(chunks, "public List<Transaction> getTransactions()")
	if getMethod == nil {
		t.Fatal("expected a method chunk for getTransactions")
	}
	if getMethod.ChunkType != string(codeindexdomain.ChunkTypeCodeBlock) {
		t.Errorf("expected code_block chunk type, got %q", getMethod.ChunkType)
	}
	if getMethod.StartLine != 8 || getMethod.EndLine != 10 {
		t.Errorf("expected getTransactions on lines 8-10, got %d-%d", getMethod.StartLine, getMethod.EndLine)
	}

	createMethod := findChunkContaining(chunks, "public Transaction createTransaction")
	if createMethod == nil || createMethod.StartLine != 13 || createMethod.EndLine != 15 {
		t.Errorf("expected createTransaction on lines 13-15, got %+v", createMethod)
	}
}
