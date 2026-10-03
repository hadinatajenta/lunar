package infrastructure

import (
	"encoding/json"
	"fmt"
	"strings"

	"lunar/backend/internal/copilot/domain"
)

func encodeChatSources(sources []domain.ChatSource) (string, error) {
	if len(sources) == 0 {
		return "", nil
	}
	encoded, err := json.Marshal(sources)
	if err != nil {
		return "", fmt.Errorf("encodeChatSources: %w", err)
	}
	return string(encoded), nil
}

func decodeChatSources(raw string) ([]domain.ChatSource, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var sources []domain.ChatSource
	if err := json.Unmarshal([]byte(raw), &sources); err != nil {
		return nil, fmt.Errorf("decodeChatSources: %w", err)
	}
	if len(sources) == 0 {
		return nil, nil
	}
	return sources, nil
}
