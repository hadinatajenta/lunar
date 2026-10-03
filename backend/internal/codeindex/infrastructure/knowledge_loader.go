package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"lunar/backend/internal/codeindex/domain"
)

const (
	knowledgeDirectoryEnv  = "LUNAR_KNOWLEDGE_DIR"
	lunarDirectoryName     = ".lunar"
	knowledgeDirectoryName = "knowledge"
	glossaryFileName       = "glossary.yaml"
)

type KnowledgeLoader struct {
	customPath string
}

func NewKnowledgeLoader(customPath ...string) *KnowledgeLoader {
	configuredPath := ""
	if len(customPath) > 0 {
		configuredPath = customPath[0]
	}
	return &KnowledgeLoader{customPath: configuredPath}
}

func ResolveKnowledgeDir() (string, error) {
	configuredPath := strings.TrimSpace(os.Getenv(knowledgeDirectoryEnv))
	if configuredPath != "" {
		absolutePath, err := filepath.Abs(configuredPath)
		if err != nil {
			return "", fmt.Errorf("ResolveKnowledgeDir: resolve %s value: %w", knowledgeDirectoryEnv, err)
		}
		return absolutePath, nil
	}

	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ResolveKnowledgeDir: resolve home directory: %w", err)
	}
	return filepath.Join(homeDirectory, lunarDirectoryName, knowledgeDirectoryName), nil
}

func ResolveGlossaryPath(customPath string) (string, error) {
	trimmedPath := strings.TrimSpace(customPath)
	if trimmedPath != "" {
		return trimmedPath, nil
	}

	knowledgeDirectory, err := ResolveKnowledgeDir()
	if err != nil {
		return "", fmt.Errorf("ResolveGlossaryPath: %w", err)
	}
	return filepath.Join(knowledgeDirectory, glossaryFileName), nil
}

func LoadGlossary(path string) (*domain.Glossary, error) {
	resolvedPath, err := ResolveGlossaryPath(path)
	if err != nil {
		return nil, err
	}

	contents, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("LoadGlossary: read %s: %w", resolvedPath, err)
	}

	glossary := &domain.Glossary{}
	if err := yaml.Unmarshal(contents, glossary); err != nil {
		return nil, fmt.Errorf("LoadGlossary: parse %s: %w", resolvedPath, err)
	}
	return glossary, nil
}

func loadGeneratedGlossary(manualGlossaryPath string) (*domain.Glossary, error) {
	generatedPath := ResolveGeneratedGlossaryPath(manualGlossaryPath)
	if _, err := os.Stat(generatedPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("loadGeneratedGlossary: inspect %s: %w", generatedPath, err)
	}

	generated, err := LoadGlossary(generatedPath)
	if err != nil {
		return nil, fmt.Errorf("loadGeneratedGlossary: %w", err)
	}
	return generated, nil
}

func (k *KnowledgeLoader) LoadAndIndex(ctx context.Context, store domain.IndexStore) (*domain.Glossary, error) {
	glossaryPath, err := ResolveGlossaryPath(k.customPath)
	if err != nil {
		return nil, fmt.Errorf("KnowledgeLoader.LoadAndIndex: %w", err)
	}

	glossary, err := LoadGlossary(glossaryPath)
	if err != nil {
		return nil, fmt.Errorf("KnowledgeLoader.LoadAndIndex: %w", err)
	}

	markdownItems, err := LoadMarkdownDocuments(filepath.Dir(glossaryPath))
	if err != nil {
		return nil, fmt.Errorf("KnowledgeLoader.LoadAndIndex: %w", err)
	}
	glossary.Notes = append(glossary.Notes, markdownItems...)

	generated, err := loadGeneratedGlossary(glossaryPath)
	if err != nil {
		return nil, fmt.Errorf("KnowledgeLoader.LoadAndIndex: %w", err)
	}

	merged := MergeGlossaries(glossary, generated)
	if err := IndexNotes(ctx, merged, store); err != nil {
		return nil, fmt.Errorf("KnowledgeLoader.LoadAndIndex: %w", err)
	}
	return merged, nil
}

func IndexNotes(ctx context.Context, glossary *domain.Glossary, store domain.IndexStore) error {
	if store == nil {
		return errors.New("IndexNotes: store must not be nil")
	}

	if err := store.ClearKnowledge(ctx); err != nil {
		return fmt.Errorf("IndexNotes: clear existing knowledge: %w", err)
	}

	items := buildKnowledgeItems(glossary)
	if len(items) == 0 {
		return nil
	}

	if err := store.SaveKnowledge(ctx, items); err != nil {
		return fmt.Errorf("IndexNotes: save knowledge: %w", err)
	}
	return nil
}

func buildKnowledgeItems(glossary *domain.Glossary) []domain.DomainKnowledgeItem {
	if glossary == nil {
		return nil
	}

	items := make([]domain.DomainKnowledgeItem, 0, len(glossary.Notes))
	for _, note := range glossary.Notes {
		term := strings.TrimSpace(note.Term)
		explanation := strings.TrimSpace(note.Explanation)
		if term == "" && explanation == "" {
			continue
		}
		items = append(items, domain.DomainKnowledgeItem{
			Term:        expandIdentifierTerm(term),
			Explanation: explanation,
		})
	}
	return items
}

func expandIdentifierTerm(term string) string {
	if term == "" || strings.ContainsAny(term, " \t\n:/\\") {
		return term
	}
	parts := splitIdentifier(term)
	if len(parts) <= 1 {
		return term
	}
	return fmt.Sprintf("%s (%s)", term, strings.Join(parts, " "))
}

func LoadMarkdownDocuments(knowledgeDirectory string) ([]domain.GlossaryNote, error) {
	items := make([]domain.GlossaryNote, 0)

	walkErr := filepath.WalkDir(knowledgeDirectory, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !isMarkdownFile(entry.Name()) {
			return nil
		}

		contents, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read markdown document %s: %w", filePath, err)
		}
		relativePath, err := filepath.Rel(knowledgeDirectory, filePath)
		if err != nil {
			return fmt.Errorf("resolve markdown document name %s: %w", filePath, err)
		}

		baseName := strings.TrimSuffix(filepath.ToSlash(relativePath), filepath.Ext(entry.Name()))
		items = append(items, parseMarkdownSections(baseName, string(contents))...)
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("LoadMarkdownDocuments: %w", walkErr)
	}
	return items, nil
}

func isMarkdownFile(name string) bool {
	extension := strings.ToLower(filepath.Ext(name))
	return extension == ".md" || extension == ".markdown"
}

func parseMarkdownSections(baseName, content string) []domain.GlossaryNote {
	items := make([]domain.GlossaryNote, 0)
	currentHeading := ""
	currentBody := &strings.Builder{}

	flushSection := func() {
		body := strings.TrimSpace(currentBody.String())
		heading := strings.TrimSpace(currentHeading)
		if heading == "" && body == "" {
			return
		}
		term := baseName
		if heading != "" {
			term = fmt.Sprintf("%s: %s", baseName, heading)
		}
		items = append(items, domain.GlossaryNote{Term: term, Explanation: body})
		currentHeading = ""
		currentBody.Reset()
	}

	for _, line := range strings.Split(content, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if isMarkdownHeading(trimmedLine) {
			flushSection()
			currentHeading = strings.TrimSpace(strings.TrimLeft(trimmedLine, "# "))
			continue
		}
		currentBody.WriteString(line)
		currentBody.WriteString("\n")
	}
	flushSection()

	return items
}

func isMarkdownHeading(line string) bool {
	return strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ")
}

func MergeGlossaries(manual, generated *domain.Glossary) *domain.Glossary {
	merged := &domain.Glossary{
		Enums: map[string]domain.GlossaryEnum{},
		Notes: make([]domain.GlossaryNote, 0),
	}

	if manual != nil {
		for name, enumDefinition := range manual.Enums {
			merged.Enums[name] = enumDefinition
		}
		merged.Notes = append(merged.Notes, manual.Notes...)
	}

	seenTerms := make(map[string]bool, len(merged.Notes))
	for _, note := range merged.Notes {
		seenTerms[normalizeNoteTerm(note.Term)] = true
	}

	if generated != nil {
		for _, note := range generated.Notes {
			normalizedTerm := normalizeNoteTerm(note.Term)
			if normalizedTerm == "" || seenTerms[normalizedTerm] {
				continue
			}
			seenTerms[normalizedTerm] = true
			merged.Notes = append(merged.Notes, note)
		}
	}

	return merged
}

func normalizeNoteTerm(term string) string {
	return strings.ToLower(strings.TrimSpace(term))
}
