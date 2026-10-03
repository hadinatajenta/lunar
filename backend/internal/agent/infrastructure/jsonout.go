package infrastructure

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	jsonFencePrefix = "```json"
	plainFence      = "```"
)

func cleanJSONResponse(rawContent string) string {
	cleaned := strings.TrimSpace(rawContent)
	lowered := strings.ToLower(cleaned)
	switch {
	case strings.HasPrefix(lowered, jsonFencePrefix):
		cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned[len(jsonFencePrefix):], plainFence))
	case strings.HasPrefix(cleaned, plainFence):
		cleaned = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(cleaned, plainFence), plainFence))
	}
	return cleaned
}

func ParseJSON[T any](raw string, repair func(string) string, validate func(T) error) (T, error) {
	var zero T
	cleaned := cleanJSONResponse(raw)
	if strings.TrimSpace(cleaned) == "" {
		return zero, errors.New("parse json: empty response")
	}

	var target T
	err := json.Unmarshal([]byte(cleaned), &target)
	if err != nil && repair != nil {
		repaired := repair(cleaned)
		if repaired != cleaned {
			err = json.Unmarshal([]byte(repaired), &target)
		}
	}
	if err != nil {
		return zero, fmt.Errorf("parse json: %w", err)
	}

	if validate != nil {
		if err := validate(target); err != nil {
			return zero, fmt.Errorf("parse json: validation failed: %w", err)
		}
	}
	return target, nil
}

type ProviderError struct {
	Provider   string
	StatusCode int
	Code       string
	Message    string
	Err        error
}

func (e *ProviderError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s API error (HTTP %d, code %s): %s", e.Provider, e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("%s API error (HTTP %d): %s", e.Provider, e.StatusCode, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

func parseProviderError(provider string, statusCode int, body []byte) *ProviderError {
	providerError := &ProviderError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    strings.TrimSpace(string(body)),
	}

	var errorEnvelope struct {
		Error struct {
			Message string          `json:"message"`
			Type    string          `json:"type"`
			Code    json.RawMessage `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorEnvelope); err == nil {
		if errorEnvelope.Error.Message != "" {
			providerError.Message = errorEnvelope.Error.Message
		}
		switch {
		case len(errorEnvelope.Error.Code) > 0:
			var codeValue string
			if err := json.Unmarshal(errorEnvelope.Error.Code, &codeValue); err == nil {
				providerError.Code = codeValue
			} else {
				providerError.Code = string(errorEnvelope.Error.Code)
			}
		case errorEnvelope.Error.Type != "":
			providerError.Code = errorEnvelope.Error.Type
		}
	}

	providerError.Err = fmt.Errorf("%s: %s", provider, providerError.Message)
	return providerError
}

func isQuotaOrRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	var providerError *ProviderError
	if errors.As(err, &providerError) {
		if providerError.StatusCode == http.StatusPaymentRequired || providerError.StatusCode == http.StatusTooManyRequests {
			return true
		}
		code := strings.ToLower(providerError.Code)
		if containsAny(code, "quota", "insufficient", "rate_limit", "resource_exhausted") {
			return true
		}
		message := strings.ToLower(providerError.Message)
		return containsAny(message, "insufficient quota", "insufficient balance", "quota exceeded", "rate limit", "resource has been exhausted")
	}

	message := strings.ToLower(err.Error())
	return containsAny(message, "http 402", "http 429", "insufficient_quota", "insufficient balance", "rate_limit", "resource_exhausted")
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
