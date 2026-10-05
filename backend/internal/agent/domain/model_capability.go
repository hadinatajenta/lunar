package domain

import "strings"

func ModelAcceptsImages(actualModel string) bool {
	switch strings.ToLower(strings.TrimSpace(actualModel)) {
	case "deepseek-flash":
		return true
	case "gpt-4o":
		return true
	case "claude-3-5-sonnet-20241022", "claude-3-7-sonnet-20250219":
		return true
	case "gemini-1.5-pro", "gemini-2.0-flash-thinking-exp":
		return true
	default:
		return false
	}
}
