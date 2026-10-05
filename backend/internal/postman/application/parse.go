package application

import (
	"encoding/json"
	"fmt"
	"strings"

	"lunar/backend/internal/postman/domain"
)

const codeFence = "```"

func StripCodeFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, codeFence) {
		return trimmed
	}

	lineEnd := strings.Index(trimmed, "\n")
	if lineEnd < 0 {
		return strings.TrimSpace(strings.Trim(trimmed, "`"))
	}

	remainder := trimmed[lineEnd+1:]
	if closing := strings.LastIndex(remainder, codeFence); closing >= 0 {
		remainder = remainder[:closing]
	}
	return strings.TrimSpace(remainder)
}

func ParseCollection(raw string, service string) (*domain.Collection, error) {
	unwrapped := StripCodeFence(raw)
	if unwrapped == "" {
		return nil, fmt.Errorf("%w: the model returned an empty response", domain.ErrCollectionInvalid)
	}

	var collection domain.Collection
	if err := json.Unmarshal([]byte(unwrapped), &collection); err != nil {
		return nil, fmt.Errorf("%w: the model response is not a Postman collection: %w", domain.ErrCollectionInvalid, err)
	}

	Normalize(&collection, service)
	if err := Validate(&collection); err != nil {
		return nil, err
	}
	return &collection, nil
}
