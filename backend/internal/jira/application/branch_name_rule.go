package application

import (
	"strings"
	"unicode"
)

const (
	defaultCoreWordLimit = 5
	intentMarker         = "i want to"
	wordSeparator        = "-"
)

var fillerWords = map[string]struct{}{
	"agar": {}, "akan": {}, "adalah": {}, "anda": {}, "bahwa": {}, "bisa": {}, "dalam": {},
	"dapat": {}, "dari": {}, "dengan": {}, "di": {}, "dan": {}, "harus": {}, "ini": {},
	"itu": {}, "juga": {}, "ke": {}, "kepada": {}, "kami": {}, "kita": {}, "memiliki": {},
	"mengajukan": {}, "menggunakan": {}, "mereka": {}, "pada": {}, "saya": {}, "secara": {},
	"sebagai": {}, "serta": {}, "sudah": {}, "terhadap": {}, "tidak": {}, "untuk": {},
	"yang": {}, "a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {},
	"by": {}, "can": {}, "for": {}, "from": {}, "i": {}, "in": {}, "is": {}, "it": {},
	"my": {}, "of": {}, "on": {}, "or": {}, "our": {}, "should": {}, "that": {}, "the": {},
	"their": {}, "this": {}, "to": {}, "want": {}, "we": {}, "will": {}, "with": {},
	"would": {},
}

func intentClause(summary string) string {
	trimmed := strings.TrimSpace(summary)
	lowercase := strings.ToLower(trimmed)
	position := strings.Index(lowercase, intentMarker)
	if position < 0 {
		return trimmed
	}

	return strings.TrimSpace(trimmed[position+len(intentMarker):])
}

func nameWords(text string) []string {
	fields := strings.FieldsFunc(text, func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsDigit(character)
	})

	words := make([]string, 0, len(fields))
	for _, field := range fields {
		words = append(words, titleWord(field))
	}

	return words
}

func titleWord(word string) string {
	runes := []rune(word)
	if len(runes) == 0 {
		return ""
	}
	if string(runes) == strings.ToUpper(word) {
		return word
	}

	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}

func withoutFillers(words []string) []string {
	meaningful := make([]string, 0, len(words))
	for _, word := range words {
		if _, isFiller := fillerWords[strings.ToLower(word)]; isFiller {
			continue
		}
		meaningful = append(meaningful, word)
	}

	return meaningful
}

func firstWords(words []string, limit int) []string {
	if limit <= 0 || len(words) <= limit {
		return words
	}

	return words[:limit]
}

func joinWords(words []string) string {
	return strings.Join(words, wordSeparator)
}

func coreFromText(text string, limit int) string {
	return joinWords(firstWords(withoutFillers(nameWords(text)), limit))
}

func fallbackCore(summary string) (string, string) {
	clause := intentClause(summary)
	core := coreFromText(clause, defaultCoreWordLimit)
	if core == "" {
		core = coreFromText(summary, defaultCoreWordLimit)
	}

	return core, clause
}
