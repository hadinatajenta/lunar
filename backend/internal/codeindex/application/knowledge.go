package application

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"lunar/backend/internal/codeindex/domain"
)

type KnowledgeLoader interface {
	LoadAndIndex(ctx context.Context, store domain.IndexStore) (*domain.Glossary, error)
}

type GlossaryGenerator interface {
	Generate(chunks map[string][]domain.CodeChunk) (int, error)
}

func BuildEnumPromptSection(glossary *domain.Glossary) string {
	if glossary == nil || len(glossary.Enums) == 0 {
		return ""
	}

	enumNames := make([]string, 0, len(glossary.Enums))
	for name := range glossary.Enums {
		enumNames = append(enumNames, name)
	}
	slices.Sort(enumNames)

	section := &strings.Builder{}
	for _, name := range enumNames {
		enumDefinition := glossary.Enums[name]
		if len(enumDefinition.Values) == 0 {
			continue
		}
		description := strings.TrimSpace(enumDefinition.Description)
		writeEnumPromptLine(section, name, description, sortedEnumValuePairs(enumDefinition.Values))
	}
	return strings.TrimRight(section.String(), "\n")
}

func sortedEnumValuePairs(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, compareEnumValueKeys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return strings.Join(pairs, ", ")
}

func compareEnumValueKeys(left, right string) int {
	leftNumber, leftErr := strconv.Atoi(left)
	rightNumber, rightErr := strconv.Atoi(right)
	if leftErr == nil && rightErr == nil {
		return cmp.Compare(leftNumber, rightNumber)
	}
	return strings.Compare(left, right)
}

func writeEnumPromptLine(section *strings.Builder, name, description, values string) {
	if description != "" {
		fmt.Fprintf(section, "- %s (%s): %s\n", name, description, values)
		return
	}
	fmt.Fprintf(section, "- %s: %s\n", name, values)
}
