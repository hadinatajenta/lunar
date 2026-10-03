package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	authApp "lunar/backend/internal/auth/application"
	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	"lunar/backend/internal/copilot/domain"
	"lunar/backend/internal/copilot/infrastructure"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type CopilotService struct {
	repo            domain.CopilotRepository
	llmClient       *infrastructure.LLMClient
	authService     *authApp.AuthService
	systemProviders map[string]string
}

func NewCopilotService(
	repo domain.CopilotRepository,
	llmClient *infrastructure.LLMClient,
	authService *authApp.AuthService,
	systemProviders map[string]string,
) *CopilotService {
	return &CopilotService{
		repo:            repo,
		llmClient:       llmClient,
		authService:     authService,
		systemProviders: systemProviders,
	}
}

func (s *CopilotService) resolveAPIKeyForProvider(ctx context.Context, userID string, provider string) (string, error) {
	decrypted, err := s.authService.GetDecryptedSecrets(ctx, userID)
	if err != nil {
		return "", err
	}

	hasAnyConfigured := len(decrypted.AIKeys) > 0
	if !hasAnyConfigured {
		for _, sysKey := range s.systemProviders {
			if strings.TrimSpace(sysKey) != "" {
				hasAnyConfigured = true
				break
			}
		}
	}

	if !hasAnyConfigured {
		return "", fmt.Errorf("%w: no AI provider configured: please set up an API key in Settings", sharedErrors.ErrBadRequest)
	}

	if key, ok := decrypted.AIKeys[provider]; ok && strings.TrimSpace(key) != "" {
		return strings.TrimSpace(key), nil
	}

	if sysKey, ok := s.systemProviders[provider]; ok && strings.TrimSpace(sysKey) != "" {
		return strings.TrimSpace(sysKey), nil
	}

	return "", fmt.Errorf("%w: no API key configured for provider '%s': please configure your API key in Settings", sharedErrors.ErrBadRequest, provider)
}

func (s *CopilotService) Chat(ctx context.Context, userID string, req domain.ChatRequest) (*domain.ChatResponse, error) {
	cleanPrompt := strings.TrimSpace(req.Prompt)
	if cleanPrompt == "" {
		return nil, fmt.Errorf("%w: message prompt cannot be empty", sharedErrors.ErrBadRequest)
	}

	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = "DeepSeek-V4 Pro (Thinking)"
	}

	provider, _, _ := s.llmClient.ResolveProviderAndModel(modelName)
	apiKey, err := s.resolveAPIKeyForProvider(ctx, userID, provider)
	if err != nil {
		return nil, err
	}

	sessionID := strings.TrimSpace(req.SessionID)
	now := time.Now().UTC()

	var session *domain.ChatSession
	if sessionID != "" {
		existing, err := s.repo.GetSessionByID(ctx, sessionID)
		if err == nil && existing != nil && existing.UserID == userID {
			session = existing
		}
	}

	if session == nil {
		title := cleanPrompt
		if len(title) > 40 {
			title = title[:40] + "..."
		}
		newID := fmt.Sprintf("session-%d", time.Now().UnixNano())
		session = &domain.ChatSession{
			ID:              newID,
			UserID:          userID,
			Title:           title,
			Model:           modelName,
			ThinkingMode:    req.ThinkingMode,
			ReasoningEffort: req.ReasoningEffort,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := s.repo.CreateSession(ctx, session); err != nil {
			return nil, err
		}
	} else {
		session.Model = modelName
		session.UpdatedAt = now
		_ = s.repo.UpdateSession(ctx, session)
	}

	userMsg := &domain.ChatMessage{
		ID:        fmt.Sprintf("msg-user-%d", time.Now().UnixNano()),
		ChatID:    session.ID,
		Role:      "user",
		Content:   cleanPrompt,
		CreatedAt: now,
	}
	if err := s.repo.SaveMessage(ctx, userMsg); err != nil {
		return nil, err
	}

	toolsContext := "none"
	if len(req.ActiveTools) > 0 {
		toolsContext = strings.Join(req.ActiveTools, ", ")
	}

	systemPrompt := fmt.Sprintf(
		"You are Lunar Copilot, a senior engineering workspace assistant for enterprise developers. "+
			"Current active workspace tools: [%s]. Provide concise, accurate technical answers. "+
			"Do not use filler phrases. Synthesize code, API contracts, and architectural facts directly.",
		toolsContext,
	)

	startLLM := time.Now()
	content, reasoning, err := s.llmClient.ChatCompletion(
		ctx,
		provider,
		apiKey,
		modelName,
		cleanPrompt,
		systemPrompt,
		req.ThinkingMode,
		req.ReasoningEffort,
	)
	if err != nil {
		return nil, err
	}
	durationMs := int(time.Since(startLLM).Milliseconds())

	sources := make([]domain.ChatSource, 0)
	for _, tool := range req.ActiveTools {
		lowerTool := strings.ToLower(tool)
		if strings.Contains(lowerTool, "jira") {
			sources = append(sources, domain.ChatSource{Type: "jira", Label: "Jira BRI · Active Issues"})
		} else if strings.Contains(lowerTool, "bitbucket") {
			sources = append(sources, domain.ChatSource{Type: "bitbucket", Label: "Bitbucket BRI · PRs & Diffs"})
		} else if strings.Contains(lowerTool, "confluence") {
			sources = append(sources, domain.ChatSource{Type: "confluence", Label: "Confluence BRI · Specifications"})
		} else if strings.Contains(lowerTool, "servicemap") {
			sources = append(sources, domain.ChatSource{Type: "servicemap", Label: "ServiceMap · Architecture Graph"})
		}
	}

	assistantMsg := &domain.ChatMessage{
		ID:                 fmt.Sprintf("msg-ast-%d", time.Now().UnixNano()),
		ChatID:             session.ID,
		Role:               "assistant",
		Content:            content,
		Reasoning:          reasoning,
		ThinkingDurationMs: durationMs,
		Sources:            sources,
		CreatedAt:          time.Now().UTC(),
	}
	if err := s.repo.SaveMessage(ctx, assistantMsg); err != nil {
		return nil, err
	}

	return &domain.ChatResponse{
		SessionID: session.ID,
		Message:   *assistantMsg,
	}, nil
}

func (s *CopilotService) ReviewPullRequest(
	ctx context.Context,
	userID string,
	repo string,
	prID string,
	diffText string,
	modelName string,
) (*bitbucketDomain.AICodeReview, error) {
	if strings.TrimSpace(diffText) == "" {
		return nil, fmt.Errorf("%w: cannot review empty git diff", sharedErrors.ErrBadRequest)
	}

	if strings.TrimSpace(modelName) == "" {
		modelName = "DeepSeek-V4 Pro (Thinking)"
	}

	provider, _, _ := s.llmClient.ResolveProviderAndModel(modelName)
	apiKey, err := s.resolveAPIKeyForProvider(ctx, userID, provider)
	if err != nil {
		return nil, err
	}

	return s.llmClient.CodeReview(ctx, provider, apiKey, modelName, repo, prID, diffText)
}

func (s *CopilotService) GetSessions(ctx context.Context, userID string) ([]domain.ChatSession, error) {
	return s.repo.GetSessionsByUserID(ctx, userID)
}

func (s *CopilotService) DeleteSession(ctx context.Context, userID string, sessionID string) error {
	return s.repo.DeleteSession(ctx, userID, sessionID)
}

func (s *CopilotService) GetMessages(ctx context.Context, userID string, sessionID string) ([]domain.ChatMessage, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, sharedErrors.ErrForbidden
	}
	return s.repo.GetMessagesByChatID(ctx, sessionID)
}
