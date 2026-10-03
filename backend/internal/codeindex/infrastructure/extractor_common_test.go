package infrastructure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

func writeFixtureFile(t *testing.T, name string, content string) (string, string) {
	t.Helper()

	fixturePath := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(fixturePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}

	readBack, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return fixturePath, string(readBack)
}

func assertChunkRangesMatchSource(t *testing.T, sourcePath string, chunks []codeindexdomain.CodeChunk) {
	t.Helper()

	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source %s: %v", sourcePath, err)
	}
	fixtureLines := strings.Split(string(sourceBytes), "\n")

	if len(chunks) == 0 {
		t.Fatalf("expected chunks for %s, got none", sourcePath)
	}

	for _, chunk := range chunks {
		if chunk.StartLine < 1 || chunk.EndLine < chunk.StartLine || chunk.EndLine > len(fixtureLines) {
			t.Fatalf(
				"chunk %s in %s has invalid range %d-%d for %d lines",
				chunk.ChunkType,
				chunk.FilePath,
				chunk.StartLine,
				chunk.EndLine,
				len(fixtureLines),
			)
		}
		slicedSource := strings.Join(fixtureLines[chunk.StartLine-1:chunk.EndLine], "\n")
		if slicedSource != chunk.RawContent {
			t.Errorf(
				"chunk %s range %d-%d does not match source slice:\nsource: %q\nchunk:  %q",
				chunk.ChunkType,
				chunk.StartLine,
				chunk.EndLine,
				slicedSource,
				chunk.RawContent,
			)
		}
	}
}

func findChunkContaining(chunks []codeindexdomain.CodeChunk, fragment string) *codeindexdomain.CodeChunk {
	for index := range chunks {
		if strings.Contains(chunks[index].RawContent, fragment) {
			return &chunks[index]
		}
	}
	return nil
}

func findChunkWithTrimmedContent(chunks []codeindexdomain.CodeChunk, trimmedContent string) *codeindexdomain.CodeChunk {
	for index := range chunks {
		if strings.TrimSpace(chunks[index].RawContent) == trimmedContent {
			return &chunks[index]
		}
	}
	return nil
}

func countChunksOfType(chunks []codeindexdomain.CodeChunk, chunkType string) int {
	count := 0
	for _, chunk := range chunks {
		if chunk.ChunkType == chunkType {
			count++
		}
	}
	return count
}

func TestFindMatchingBraceSkipsStringsAndComments(t *testing.T) {
	fixture := `{
	$first = "brace } inside double quotes";
	$second = 'brace } inside single quotes';
	$escaped = "escaped \" and brace \}";
	$closing = 1;
}`
	closeIndex, isFound := findMatchingBrace(fixture, 0)
	if !isFound {
		t.Fatal("expected findMatchingBrace to locate the closing brace")
	}
	if closeIndex != len(fixture)-1 {
		t.Errorf("expected closing brace at %d, got %d", len(fixture)-1, closeIndex)
	}
}

func TestFindNextOpenBraceRejectsBodylessSignature(t *testing.T) {
	bodyless := "void run();"
	if openIndex := findNextOpenBrace(bodyless, 0); openIndex != -1 {
		t.Errorf("expected -1 for bodyless signature, got %d", openIndex)
	}

	withBody := "void run() { return; }"
	openIndex := findNextOpenBrace(withBody, 0)
	if openIndex < 0 || withBody[openIndex] != '{' {
		t.Errorf("expected brace index for concrete function, got %d", openIndex)
	}
}

func TestExtractFileDispatchesByExtension(t *testing.T) {
	fixture := "package main\n\nfunc main() {\n}\n"
	filePath, content := writeFixtureFile(t, "main.go", fixture)

	chunks, _ := ExtractFile(content, "main.go", "sample-service", filepath.Ext(filePath))
	if len(chunks) == 0 {
		t.Fatal("expected Go chunks from ExtractFile dispatch")
	}

	unsupported, _ := ExtractFile(content, "main.rb", "sample-service", ".rb")
	if len(unsupported) != 0 {
		t.Errorf("expected no chunks for unsupported extension, got %d", len(unsupported))
	}
}
