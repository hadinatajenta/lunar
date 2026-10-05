package application

import (
	"strings"

	"lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

func buildUserMessage(question codeindexdomain.AgentQuestion) domain.ChatMessage {
	text := question.Question
	if block := strings.TrimSpace(question.AttachmentBlock); block != "" {
		text = block + "\n\n" + text
	}

	if len(question.ImageDataURLs) == 0 {
		return domain.ChatMessage{Role: userRole, Content: domain.TextMessage(text)}
	}

	parts := make([]domain.ContentPart, 0, len(question.ImageDataURLs)+1)
	parts = append(parts, domain.TextPart{Type: domain.ContentPartTypeText, Text: text})
	for _, dataURL := range question.ImageDataURLs {
		parts = append(parts, domain.ImagePart{
			Type:     domain.ContentPartTypeImage,
			ImageURL: domain.ImageURL{URL: dataURL},
		})
	}
	return domain.ChatMessage{Role: userRole, Content: domain.MessageContent{Parts: parts}}
}
