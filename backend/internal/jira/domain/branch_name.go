package domain

import "strings"

const (
	BranchNameSourceModel    = "model"
	BranchNameSourceFallback = "fallback"

	branchNamePrefix = "feature/"
)

type BranchNameSuggestion struct {
	Key            string `json:"key"`
	Branch         string `json:"branch"`
	Category       string `json:"category"`
	Core           string `json:"core"`
	Source         string `json:"source"`
	Evidence       string `json:"evidence"`
	EvidenceSource string `json:"evidence_source"`
}

func BuildBranchName(key string, category string, core string) string {
	parts := make([]string, 0, 3)
	parts = append(parts, key)
	if category != "" {
		parts = append(parts, category)
	}
	if core != "" {
		parts = append(parts, core)
	}

	return branchNamePrefix + strings.Join(parts, "-")
}
