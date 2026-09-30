package domain

import (
	"strings"
)

func ExtractSquadName(sprintName string) string {
	clean := strings.TrimSpace(sprintName)
	if clean == "" {
		return ""
	}

	if strings.Contains(clean, " - ") {
		parts := strings.Split(clean, " - ")
		if strings.HasPrefix(strings.ToLower(parts[0]), "sprint") {
			return strings.TrimSpace(parts[1])
		}
		if strings.HasPrefix(strings.ToLower(parts[1]), "sprint") {
			return strings.TrimSpace(parts[0])
		}
	}

	if strings.Contains(clean, "-") {
		parts := strings.Split(clean, "-")
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(parts[0])), "sprint") {
			return strings.TrimSpace(parts[1])
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(parts[1])), "sprint") {
			return strings.TrimSpace(parts[0])
		}
	}

	if strings.Contains(clean, "#") {
		parts := strings.Split(clean, "#")
		candidate := strings.TrimSpace(parts[0])
		lower := strings.ToLower(candidate)
		if strings.HasPrefix(lower, "sprint ") {
			return strings.TrimSpace(candidate[7:])
		}
		return candidate
	}

	return clean
}

func MatchSquad(sprintName string, targetSquad string) bool {
	cleanTarget := strings.TrimSpace(targetSquad)
	if cleanTarget == "" || strings.EqualFold(cleanTarget, "all") {
		return true
	}

	extracted := ExtractSquadName(sprintName)
	if strings.EqualFold(extracted, cleanTarget) {
		return true
	}

	lowerExtracted := strings.ToLower(extracted)
	lowerTarget := strings.ToLower(cleanTarget)

	if strings.Contains(lowerExtracted, lowerTarget) || strings.Contains(lowerTarget, lowerExtracted) {
		return true
	}

	trimmedExtracted := strings.TrimSuffix(lowerExtracted, " squad")
	trimmedTarget := strings.TrimSuffix(lowerTarget, " squad")
	if trimmedExtracted == trimmedTarget {
		return true
	}

	if trimmedExtracted != "" && trimmedTarget != "" {
		if strings.Contains(trimmedExtracted, trimmedTarget) || strings.Contains(trimmedTarget, trimmedExtracted) {
			return true
		}
	}

	return false
}
