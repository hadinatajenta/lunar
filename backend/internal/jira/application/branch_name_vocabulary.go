package application

import (
	"context"
	"sort"
	"strings"
)

const maximumObservedCategories = 12

type CategoryVocabularySource interface {
	ObserveCategories(ctx context.Context, userID string) (map[string]int, error)
}

type ObservedCategory struct {
	Name  string
	Count int
}

type CategoryVocabulary struct {
	Listed        []ObservedCategory
	ObservedTotal int
}

func BuildCategoryVocabulary(counts map[string]int) CategoryVocabulary {
	observed := mergedCategoryCounts(counts)
	sort.Slice(observed, func(first int, second int) bool {
		if observed[first].Count != observed[second].Count {
			return observed[first].Count > observed[second].Count
		}

		return observed[first].Name < observed[second].Name
	})

	listed := observed
	if len(listed) > maximumObservedCategories {
		listed = listed[:maximumObservedCategories]
	}

	return CategoryVocabulary{Listed: listed, ObservedTotal: len(observed)}
}

func (v CategoryVocabulary) IsEmpty() bool {
	return len(v.Listed) == 0
}

func (v CategoryVocabulary) Resolve(category string) (string, bool) {
	normalised, err := normaliseNameSegment(category, maximumCategoryWords)
	if err != nil {
		return "", false
	}

	for _, observed := range v.Listed {
		if strings.EqualFold(observed.Name, normalised) {
			return observed.Name, true
		}
	}

	return "", false
}

func mergedCategoryCounts(counts map[string]int) []ObservedCategory {
	merged := make(map[string]ObservedCategory, len(counts))
	for name, count := range counts {
		normalised, err := normaliseNameSegment(name, maximumCategoryWords)
		if err != nil {
			continue
		}

		key := strings.ToLower(normalised)
		observed, found := merged[key]
		if !found {
			merged[key] = ObservedCategory{Name: normalised, Count: count}
			continue
		}

		observed.Count += count
		if normalised < observed.Name {
			observed.Name = normalised
		}
		merged[key] = observed
	}

	categories := make([]ObservedCategory, 0, len(merged))
	for _, observed := range merged {
		categories = append(categories, observed)
	}

	return categories
}
