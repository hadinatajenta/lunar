package infrastructure

import (
	"path/filepath"
	"regexp"
	"strings"
)

const redactedValue = "***"

var excludedFileNames = map[string]bool{
	".env":      true,
	".npmrc":    true,
	".netrc":    true,
	".htpasswd": true,
}

var excludedFilePrefixes = []string{
	".env.",
	"id_rsa",
	"id_ed25519",
	"credentials",
}

var excludedFileSuffixes = []string{
	".pem",
	".key",
	".p12",
	".keystore",
	".jks",
}

const excludedFileKeyword = "secret"

var secretAssignmentPattern = regexp.MustCompile(
	`(?i)(api[_-]?key|secret|password|passwd|token|private[_-]?key)\s*[:=]\s*["'][^"']{8,}["']`,
)

func IsExcludedFileName(name string) bool {
	lowered := strings.ToLower(filepath.Base(name))
	if excludedFileNames[lowered] {
		return true
	}
	if strings.Contains(lowered, excludedFileKeyword) {
		return true
	}
	for _, prefix := range excludedFilePrefixes {
		if strings.HasPrefix(lowered, prefix) {
			return true
		}
	}
	for _, suffix := range excludedFileSuffixes {
		if strings.HasSuffix(lowered, suffix) {
			return true
		}
	}
	return false
}

func RedactSecrets(text string) (string, int) {
	redactionCount := 0
	redacted := secretAssignmentPattern.ReplaceAllStringFunc(text, func(match string) string {
		redactionCount++
		return redactSecretMatch(match)
	})
	return redacted, redactionCount
}

func RedactSecretsPerLine(text string) (string, int) {
	if !strings.Contains(text, "\n") {
		return RedactSecrets(text)
	}

	redactionCount := 0
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		redactedLine, lineCount := RedactSecrets(line)
		redactionCount += lineCount
		lines[index] = redactedLine
	}
	return strings.Join(lines, "\n"), redactionCount
}

func redactSecretMatch(match string) string {
	start := strings.IndexAny(match, `"'`)
	if start < 0 {
		return match
	}
	end := strings.LastIndexAny(match, `"'`)
	if end <= start {
		return match
	}
	return match[:start+1] + redactedValue + match[end:]
}
