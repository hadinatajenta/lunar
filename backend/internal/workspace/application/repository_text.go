package application

import (
	"fmt"
	"sort"
	"strings"
)

const (
	minimumTermLength   = 4
	maximumReasonLength = 300
)

var repositoryNoiseTokens = map[string]struct{}{
	"api": {}, "app": {}, "apps": {}, "backend": {}, "be": {}, "common": {}, "config": {},
	"core": {}, "docs": {}, "fe": {}, "frontend": {}, "gateway": {}, "lib": {}, "libs": {},
	"micro": {}, "microservice": {}, "microservices": {}, "mms": {}, "module": {}, "service": {},
	"services": {}, "test": {}, "tests": {}, "ui": {}, "util": {}, "utils": {}, "web": {},
}

var termStopWords = map[string]struct{}{
	"about": {}, "acceptance": {}, "after": {}, "again": {}, "akan": {}, "also": {}, "anda": {},
	"another": {}, "atau": {}, "bisa": {}, "dalam": {}, "dapat": {}, "dari": {}, "dengan": {},
	"description": {}, "field": {}, "from": {}, "have": {}, "issue": {}, "mereka": {}, "pada": {},
	"that": {}, "their": {}, "there": {}, "this": {}, "untuk": {}, "user": {}, "users": {},
	"want": {}, "when": {}, "where": {}, "which": {}, "will": {}, "with": {}, "yang": {},
}

func tokeniseText(text string) []string {
	return splitTokens(text)
}

func repositoryTokens(repositoryName string) []string {
	return splitTokens(repositoryName)
}

func splitTokens(text string) []string {
	tokens := make([]string, 0, 16)
	var current strings.Builder

	for _, character := range text {
		switch {
		case character >= 'a' && character <= 'z':
			current.WriteRune(character)
		case character >= 'A' && character <= 'Z':
			if startsNewToken(current.String()) {
				flushToken(&tokens, &current)
			}
			current.WriteRune(character + ('a' - 'A'))
		case character >= '0' && character <= '9':
			current.WriteRune(character)
		default:
			flushToken(&tokens, &current)
		}
	}

	flushToken(&tokens, &current)
	return tokens
}

func startsNewToken(current string) bool {
	if current == "" {
		return false
	}

	previous := current[len(current)-1]
	return (previous >= 'a' && previous <= 'z') || (previous >= '0' && previous <= '9')
}

func flushToken(tokens *[]string, current *strings.Builder) {
	if current.Len() == 0 {
		return
	}

	*tokens = append(*tokens, current.String())
	current.Reset()
}

func containsTokenRun(haystack []string, needle []string) bool {
	if len(needle) == 0 || len(haystack) < len(needle) {
		return false
	}

	for start := 0; start <= len(haystack)-len(needle); start++ {
		if matchesTokenRun(haystack[start:], needle) {
			return true
		}
	}

	return false
}

func matchesTokenRun(haystack []string, needle []string) bool {
	for offset, token := range needle {
		if haystack[offset] != token {
			return false
		}
	}

	return true
}

func distinctiveRepositoryToken(token string) bool {
	if len(token) < minimumTermLength {
		return false
	}

	_, isNoise := repositoryNoiseTokens[token]
	return !isNoise
}

func distinctiveTerm(token string) bool {
	if len(token) < minimumTermLength {
		return false
	}
	if _, isNoise := repositoryNoiseTokens[token]; isNoise {
		return false
	}
	if _, isStopWord := termStopWords[token]; isStopWord {
		return false
	}
	return true
}

func truncateReason(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= maximumReasonLength {
		return trimmed
	}

	return strings.TrimSpace(trimmed[:maximumReasonLength])
}
func repositoryHintStrength(repositoryName string, textTokens []string) (float64, string) {
	nameTokens := repositoryTokens(repositoryName)
	if len(nameTokens) == 0 {
		return 0, ""
	}
	if containsTokenRun(textTokens, nameTokens) {
		return fullHintStrength, strings.Join(nameTokens, "-")
	}
	if len(nameTokens) == 1 {
		return 0, ""
	}

	for _, token := range nameTokens {
		if !distinctiveRepositoryToken(token) {
			continue
		}
		if containsTokenRun(textTokens, []string{token}) {
			return partialHintStrength, token
		}
	}

	return 0, ""
}

func pageHintStrength(repositoryName string, pages []LinkedPage) (float64, string) {
	bestStrength := 0.0
	bestPage := ""
	for _, page := range pages {
		strength, _ := repositoryHintStrength(repositoryName, tokeniseText(page.Title+"\n"+page.Body))
		if strength <= bestStrength {
			continue
		}

		bestStrength = strength
		bestPage = strings.TrimSpace(page.Title)
	}

	return bestStrength, bestPage
}

func hintReason(repo string, matched string) string {
	if strings.EqualFold(strings.ReplaceAll(matched, "-", ""), strings.ReplaceAll(repo, "-", "")) {
		return fmt.Sprintf("the issue names %s", repo)
	}

	return fmt.Sprintf("the issue mentions %s", matched)
}

func codeIndexReason(repo string, termCount int, fileCount int) string {
	if fileCount <= 0 {
		return fmt.Sprintf("%d of the issue's terms appear in %s", termCount, repo)
	}

	return fmt.Sprintf("%d of the issue's terms appear in %d files in %s", termCount, fileCount, repo)
}

func distinctiveTerms(text string, limit int) []string {
	seen := make(map[string]struct{})
	terms := make([]string, 0, limit)
	for _, token := range tokeniseText(text) {
		if !distinctiveTerm(token) {
			continue
		}
		if _, duplicate := seen[token]; duplicate {
			continue
		}

		seen[token] = struct{}{}
		terms = append(terms, token)
	}

	sort.SliceStable(terms, func(first int, second int) bool {
		if len(terms[first]) != len(terms[second]) {
			return len(terms[first]) > len(terms[second])
		}
		return terms[first] < terms[second]
	})

	if len(terms) > limit {
		terms = terms[:limit]
	}

	return terms
}
