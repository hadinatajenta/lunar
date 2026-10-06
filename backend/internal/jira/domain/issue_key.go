package domain

import (
	"fmt"
	"regexp"
	"strings"
)

const IssueKeyPattern = `[A-Z][A-Z0-9]{1,9}-[0-9]{1,6}(?![0-9])`

var issueKeyExpression = regexp.MustCompile(`[A-Z][A-Z0-9]{1,9}-[0-9]{1,6}`)

var issueKeyValidationExpression = regexp.MustCompile(`(?i)^[A-Z][A-Z0-9]{1,9}-[0-9]{1,6}$`)

func ValidateIssueKey(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if !issueKeyValidationExpression.MatchString(trimmed) {
		return "", fmt.Errorf("issue key %q is not a valid Jira issue key", trimmed)
	}

	return strings.ToUpper(trimmed), nil
}

func IssueKeysIn(text string) []string {
	locations := issueKeyExpression.FindAllStringIndex(text, -1)
	keys := make([]string, 0, len(locations))
	seen := make(map[string]struct{}, len(locations))

	for _, location := range locations {
		if hasTrailingDigit(text, location[1]) {
			continue
		}

		key := strings.ToUpper(text[location[0]:location[1]])
		if _, duplicate := seen[key]; duplicate {
			continue
		}

		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	return keys
}

func ContainsIssueKey(text string, key string) bool {
	upperText := strings.ToUpper(text)
	upperKey := strings.ToUpper(strings.TrimSpace(key))
	if upperKey == "" {
		return false
	}

	searchFrom := 0
	for searchFrom <= len(upperText)-len(upperKey) {
		offset := strings.Index(upperText[searchFrom:], upperKey)
		if offset < 0 {
			return false
		}

		end := searchFrom + offset + len(upperKey)
		if !hasTrailingDigit(upperText, end) {
			return true
		}

		searchFrom = searchFrom + offset + 1
	}

	return false
}

func hasTrailingDigit(text string, end int) bool {
	if end >= len(text) {
		return false
	}

	character := text[end]
	return character >= '0' && character <= '9'
}
