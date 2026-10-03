package infrastructure

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lunar/backend/internal/codeindex/domain"
)

func newGlossaryGeneratorInTemp(t *testing.T) (*GlossaryGenerator, string) {
	t.Helper()
	directory := t.TempDir()
	manualPath := filepath.Join(directory, "glossary.yaml")
	if err := os.WriteFile(manualPath, []byte("enums: {}\nnotes: []\n"), 0o600); err != nil {
		t.Fatalf("cannot write manual glossary: %v", err)
	}
	return NewGlossaryGenerator(manualPath), directory
}

func readGeneratedArtifact(t *testing.T, directory string) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(directory, generatedGlossaryFileName))
	if err != nil {
		t.Fatalf("generated artifact not found under the temp directory: %v", err)
	}
	return contents
}

func loadGeneratedArtifact(t *testing.T, directory string) *domain.Glossary {
	t.Helper()
	glossary, err := LoadGlossary(filepath.Join(directory, generatedGlossaryFileName))
	if err != nil {
		t.Fatalf("LoadGlossary for generated artifact: %v", err)
	}
	return glossary
}

func enumChunk(repoName, filePath, rawContent string) domain.CodeChunk {
	return domain.CodeChunk{RepoName: repoName, FilePath: filePath, ChunkType: string(domain.ChunkTypeEnumDef), RawContent: rawContent}
}

func TestResolveGeneratedGlossaryPath(t *testing.T) {
	expected := filepath.Join("/a/b", generatedGlossaryFileName)
	if actual := ResolveGeneratedGlossaryPath("/a/b/glossary.yaml"); actual != expected {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

func TestGenerateUsesAnnotationKeyAndDescription(t *testing.T) {
	generator, directory := newGlossaryGeneratorInTemp(t)
	chunks := map[string][]domain.CodeChunk{
		"pacific": {enumChunk("pacific", "internal/domain/status.go", "// @enum: status_aktif - Status user aktif\nconst StatusActive = \"active\"")},
	}

	count, err := generator.Generate(chunks)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 note, got %d", count)
	}

	glossary := loadGeneratedArtifact(t, directory)
	if len(glossary.Notes) != 1 {
		t.Fatalf("expected 1 note in artifact, got %d", len(glossary.Notes))
	}

	note := glossary.Notes[0]
	if note.Term != "enum status_aktif pacific StatusActive" {
		t.Fatalf("unexpected term %q", note.Term)
	}
	if !strings.Contains(note.Explanation, "Enum candidate from source code (pacific/internal/domain/status.go). status_aktif: active = StatusActive.") {
		t.Fatalf("unexpected explanation %q", note.Explanation)
	}
	if !strings.Contains(note.Explanation, "Note: Status user aktif.") {
		t.Fatalf("annotation description missing from %q", note.Explanation)
	}
}

func TestGenerateDerivesPrefixKeyFromIotaBlock(t *testing.T) {
	generator, directory := newGlossaryGeneratorInTemp(t)
	chunks := map[string][]domain.CodeChunk{
		"pacific": {enumChunk("pacific", "internal/domain/order_status.go", "const (\n\tStatusPending = iota\n\tStatusApproved\n)")},
	}

	count, err := generator.Generate(chunks)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 note, got %d", count)
	}

	note := loadGeneratedArtifact(t, directory).Notes[0]
	if note.Term != "enum status pacific StatusPending StatusApproved" {
		t.Fatalf("unexpected term %q", note.Term)
	}
	if !strings.Contains(note.Explanation, "status: 0 = StatusPending, 1 = StatusApproved.") {
		t.Fatalf("unexpected explanation %q", note.Explanation)
	}
}

func TestGeneratePrefersTrailingCommentLabel(t *testing.T) {
	generator, directory := newGlossaryGeneratorInTemp(t)
	chunks := map[string][]domain.CodeChunk{
		"pacific": {enumChunk("pacific", "internal/domain/order_status.go", "const (\n\tStatusPending = iota // Pending\n\tStatusApproved\n)")},
	}

	count, err := generator.Generate(chunks)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 note, got %d", count)
	}

	explanation := loadGeneratedArtifact(t, directory).Notes[0].Explanation
	if !strings.Contains(explanation, "0 = Pending") {
		t.Fatalf("expected trailing comment label, got %q", explanation)
	}
	if strings.Contains(explanation, "0 = StatusPending") {
		t.Fatalf("member name must not win over the trailing comment, got %q", explanation)
	}
}

func TestGenerateLeavesArtifactUntouchedWithoutCandidates(t *testing.T) {
	generator, directory := newGlossaryGeneratorInTemp(t)
	artifactPath := filepath.Join(directory, generatedGlossaryFileName)
	sentinel := []byte("sentinel: true\n")
	if err := os.WriteFile(artifactPath, sentinel, 0o600); err != nil {
		t.Fatalf("cannot write sentinel artifact: %v", err)
	}

	chunks := map[string][]domain.CodeChunk{
		"pacific": {
			{RepoName: "pacific", FilePath: "internal/http/router.go", ChunkType: string(domain.ChunkTypeRoute), RawContent: "r.GET(\"/health\", handler)"},
			enumChunk("pacific", "internal/domain/solo.go", "const Alone = \"solo\""),
		},
	}

	count, err := generator.Generate(chunks)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 notes, got %d", count)
	}
	if contents := readGeneratedArtifact(t, directory); !bytes.Equal(contents, sentinel) {
		t.Fatalf("artifact must stay untouched, got %q", contents)
	}
}

func TestGenerateCapsTermAndExplanation(t *testing.T) {
	generator, directory := newGlossaryGeneratorInTemp(t)

	builder := &strings.Builder{}
	builder.WriteString("const (\n")
	for index := 0; index < 30; index++ {
		name := fmt.Sprintf("Status%c%c", 'A'+rune(index/26), 'A'+rune(index%26))
		if index == 0 {
			fmt.Fprintf(builder, "\t%s = iota\n", name)
			continue
		}
		fmt.Fprintf(builder, "\t%s\n", name)
	}
	builder.WriteString(")")

	chunks := map[string][]domain.CodeChunk{
		"pacific": {enumChunk("pacific", "internal/domain/cap.go", builder.String())},
	}
	if count, err := generator.Generate(chunks); err != nil || count != 1 {
		t.Fatalf("Generate returned count %d and error %v", count, err)
	}

	note := loadGeneratedArtifact(t, directory).Notes[0]
	if !strings.Contains(note.Explanation, "24 = StatusAY, ... (+5 more values)") {
		t.Fatalf("expected capped explanation with 5 remaining values, got %q", note.Explanation)
	}

	names := make([]string, 0, maxMembersInTerm)
	for index := 0; index < maxMembersInTerm; index++ {
		names = append(names, fmt.Sprintf("Status%c%c", 'A'+rune(index/26), 'A'+rune(index%26)))
	}
	expected := "enum status pacific " + strings.Join(names, " ") + " ..."
	if note.Term != expected {
		t.Fatalf("expected term %q, got %q", expected, note.Term)
	}
}

func TestGenerateArtifactIsDeterministicAndSorted(t *testing.T) {
	generator, directory := newGlossaryGeneratorInTemp(t)

	alphaZeta := enumChunk("alpha", "internal/alpha_z.go", "// @enum: alpha_zeta - Zeta alpha\nconst AlphaZeta = \"z\"")
	alphaAlpha := enumChunk("alpha", "internal/alpha_a.go", "// @enum: alpha_alpha - Alpha alpha\nconst AlphaAlpha = \"a\"")
	zebraBeta := enumChunk("zebra", "internal/zebra_b.go", "// @enum: zebra_beta - Beta zebra\nconst ZebraBeta = \"b\"")
	zebraAlpha := enumChunk("zebra", "internal/zebra_a.go", "// @enum: zebra_alpha - Alpha zebra\nconst ZebraAlpha = \"a\"")

	firstCount, err := generator.Generate(map[string][]domain.CodeChunk{
		"alpha": {alphaZeta, alphaAlpha},
		"zebra": {zebraBeta, zebraAlpha},
	})
	if err != nil {
		t.Fatalf("first Generate: %v", err)
	}
	firstArtifact := readGeneratedArtifact(t, directory)

	secondCount, err := generator.Generate(map[string][]domain.CodeChunk{
		"alpha": {alphaAlpha, alphaZeta},
		"zebra": {zebraAlpha, zebraBeta},
	})
	if err != nil {
		t.Fatalf("second Generate: %v", err)
	}
	secondArtifact := readGeneratedArtifact(t, directory)

	if firstCount != 4 || secondCount != 4 {
		t.Fatalf("expected 4 notes per run, got %d and %d", firstCount, secondCount)
	}
	if !bytes.Equal(firstArtifact, secondArtifact) {
		t.Fatalf("artifact is not deterministic:\nfirst:  %q\nsecond: %q", firstArtifact, secondArtifact)
	}

	notes := loadGeneratedArtifact(t, directory).Notes
	if len(notes) != 4 {
		t.Fatalf("expected 4 notes, got %d", len(notes))
	}
	for index := 1; index < len(notes); index++ {
		if notes[index-1].Term >= notes[index].Term {
			t.Fatalf("notes not sorted by term: %q before %q", notes[index-1].Term, notes[index].Term)
		}
	}
}

func TestGenerateArtifactRoundTripsThroughLoadGlossary(t *testing.T) {
	generator, directory := newGlossaryGeneratorInTemp(t)
	chunks := map[string][]domain.CodeChunk{
		"pacific": {enumChunk("pacific", "internal/domain/status.go", "// @enum: status_aktif - Status user aktif\nconst StatusActive = \"active\"")},
	}

	count, err := generator.Generate(chunks)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	artifact := readGeneratedArtifact(t, directory)
	if !strings.Contains(string(artifact), "enums: {}") {
		t.Fatalf("artifact must render an empty enums mapping, got:\n%s", artifact)
	}

	glossary := loadGeneratedArtifact(t, directory)
	if len(glossary.Notes) != count {
		t.Fatalf("expected %d notes after round trip, got %d", count, len(glossary.Notes))
	}
	if len(glossary.Enums) != 0 {
		t.Fatalf("expected 0 enums after round trip, got %d", len(glossary.Enums))
	}
}
