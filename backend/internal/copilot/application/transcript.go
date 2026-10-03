package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"lunar/backend/internal/copilot/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func (s *CopilotService) SaveTranscript(
	ctx context.Context,
	userID string,
	req domain.TranscriptRequest,
) (*domain.TranscriptResponse, error) {
	sessionID, question, answer, err := validateTranscriptRequest(req)
	if err != nil {
		return nil, err
	}

	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFound) {
			return nil, sharedErrors.ErrNotFound
		}
		return nil, fmt.Errorf("saveTranscript: load session: %w", err)
	}
	if session.UserID != userID {
		return nil, sharedErrors.ErrForbidden
	}

	session.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("saveTranscript: update session: %w", err)
	}

	questionMessage := &domain.ChatMessage{
		ID:        fmt.Sprintf("msg-user-%d", time.Now().UnixNano()),
		ChatID:    session.ID,
		Role:      "user",
		Content:   question,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.SaveMessage(ctx, questionMessage); err != nil {
		return nil, fmt.Errorf("saveTranscript: save question: %w", err)
	}

	answerMessage := &domain.ChatMessage{
		ID:                 fmt.Sprintf("msg-ast-%d", time.Now().UnixNano()),
		ChatID:             session.ID,
		Role:               "assistant",
		Content:            answer,
		Reasoning:          strings.TrimSpace(req.Reasoning),
		ThinkingDurationMs: req.DurationMs,
		Sources:            mapTranscriptSources(req.Sources),
		CreatedAt:          time.Now().UTC(),
	}
	if err := s.repo.SaveMessage(ctx, answerMessage); err != nil {
		return nil, fmt.Errorf("saveTranscript: save answer: %w", err)
	}

	return &domain.TranscriptResponse{MessageID: answerMessage.ID}, nil
}

func validateTranscriptRequest(req domain.TranscriptRequest) (string, string, string, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return "", "", "", fmt.Errorf("%w: session id is required", sharedErrors.ErrBadRequest)
	}

	question := strings.TrimSpace(req.Question)
	if question == "" {
		return "", "", "", fmt.Errorf("%w: question cannot be empty", sharedErrors.ErrBadRequest)
	}

	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		return "", "", "", fmt.Errorf("%w: answer cannot be empty", sharedErrors.ErrBadRequest)
	}

	if req.DurationMs < 0 {
		return "", "", "", fmt.Errorf("%w: duration_ms cannot be negative", sharedErrors.ErrBadRequest)
	}

	return sessionID, question, answer, nil
}

func mapTranscriptSources(clientSources []domain.TranscriptSource) []domain.ChatSource {
	sources := make([]domain.ChatSource, 0, len(clientSources))
	for _, clientSource := range clientSources {
		repoName := strings.TrimSpace(clientSource.Repo)
		sourcePath := strings.TrimSpace(clientSource.File)
		if sourcePath == "" {
			sourcePath = strings.TrimSpace(clientSource.Path)
		}
		sourceType := strings.TrimSpace(clientSource.Type)
		if repoName == "" && sourcePath == "" && sourceType == "" {
			continue
		}

		sources = append(sources, domain.ChatSource{
			Type:      sourceType,
			Label:     buildSourceLabel(repoName, sourcePath),
			Repo:      repoName,
			Path:      sourcePath,
			StartLine: clientSource.StartLine,
			EndLine:   clientSource.EndLine,
		})
	}
	return sources
}

func buildSourceLabel(repoName string, sourcePath string) string {
	if repoName != "" && sourcePath != "" {
		return repoName + "/" + sourcePath
	}
	if sourcePath != "" {
		return sourcePath
	}
	return repoName
}
