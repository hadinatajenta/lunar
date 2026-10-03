package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"lunar/backend/internal/copilot/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	transcriptUserID    = "user-transcript-a"
	transcriptOtherUser = "user-transcript-b"
	transcriptSessionID = "session-transcript-1"
)

type fakeCopilotRepository struct {
	sessions   map[string]*domain.ChatSession
	messages   map[string][]domain.ChatMessage
	updateErr  error
	saveErr    error
	saveCalls  int
	updateCall int
}

func newFakeCopilotRepository() *fakeCopilotRepository {
	return &fakeCopilotRepository{
		sessions: make(map[string]*domain.ChatSession),
		messages: make(map[string][]domain.ChatMessage),
	}
}

func (f *fakeCopilotRepository) seedSession(session domain.ChatSession) {
	storedCopy := session
	f.sessions[session.ID] = &storedCopy
}

func (f *fakeCopilotRepository) GetSessionsByUserID(ctx context.Context, userID string) ([]domain.ChatSession, error) {
	return nil, nil
}

func (f *fakeCopilotRepository) GetSessionByID(ctx context.Context, sessionID string) (*domain.ChatSession, error) {
	session, exists := f.sessions[sessionID]
	if !exists {
		return nil, sharedErrors.ErrNotFound
	}
	storedCopy := *session
	return &storedCopy, nil
}

func (f *fakeCopilotRepository) CreateSession(ctx context.Context, session *domain.ChatSession) error {
	f.seedSession(*session)
	return nil
}

func (f *fakeCopilotRepository) UpdateSession(ctx context.Context, session *domain.ChatSession) error {
	f.updateCall++
	if f.updateErr != nil {
		return f.updateErr
	}
	f.seedSession(*session)
	return nil
}

func (f *fakeCopilotRepository) DeleteSession(ctx context.Context, userID string, sessionID string) error {
	delete(f.sessions, sessionID)
	return nil
}

func (f *fakeCopilotRepository) GetMessagesByChatID(ctx context.Context, chatID string) ([]domain.ChatMessage, error) {
	stored := f.messages[chatID]
	messageCopy := make([]domain.ChatMessage, len(stored))
	copy(messageCopy, stored)
	return messageCopy, nil
}

func (f *fakeCopilotRepository) SaveMessage(ctx context.Context, msg *domain.ChatMessage) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	storedCopy := *msg
	f.messages[msg.ChatID] = append(f.messages[msg.ChatID], storedCopy)
	return nil
}

func validTranscriptRequest() domain.TranscriptRequest {
	return domain.TranscriptRequest{
		SessionID:  transcriptSessionID,
		Question:   "how does transcript persistence work?",
		Answer:     "it stores a question and an answer",
		Reasoning:  "history belongs to the backend",
		DurationMs: 1234,
	}
}

func newTranscriptService(repo *fakeCopilotRepository, ownerID string, updatedAt time.Time) *CopilotService {
	repo.seedSession(domain.ChatSession{
		ID:        transcriptSessionID,
		UserID:    ownerID,
		Title:     "existing session",
		Model:     "deepseek-flash",
		CreatedAt: updatedAt,
		UpdatedAt: updatedAt,
	})
	return NewCopilotService(repo, nil, nil, nil)
}

func TestSaveTranscript_PersistsQuestionAnswerAndSources(t *testing.T) {
	repo := newFakeCopilotRepository()
	sessionUpdatedAt := time.Date(2024, time.May, 1, 10, 0, 0, 0, time.UTC)
	service := newTranscriptService(repo, transcriptUserID, sessionUpdatedAt)

	request := validTranscriptRequest()
	request.Question = "  how does transcript persistence work?  "
	request.Answer = "  it stores a question and an answer  "
	request.Reasoning = "  history belongs to the backend  "
	request.Sources = []domain.TranscriptSource{
		{Repo: "everest", File: "internal/api/handler.go", Type: "code", Snippet: "func handler() {}", StartLine: 12, EndLine: 30},
		{Path: "aurora/worker.go", Type: "code"},
		{Snippet: "source without repository, path or type"},
	}

	response, err := service.SaveTranscript(context.Background(), transcriptUserID, request)
	if err != nil {
		t.Fatalf("expected the transcript to persist, got error: %v", err)
	}
	if response == nil || response.MessageID == "" {
		t.Fatal("expected the transcript response to carry the assistant message id")
	}

	storedMessages, err := repo.GetMessagesByChatID(context.Background(), transcriptSessionID)
	if err != nil {
		t.Fatalf("cannot read stored messages: %v", err)
	}
	if len(storedMessages) != 2 {
		t.Fatalf("expected a question and an answer to be stored, got %d messages", len(storedMessages))
	}

	questionMessage := storedMessages[0]
	if questionMessage.Role != "user" || questionMessage.Content != "how does transcript persistence work?" {
		t.Errorf("unexpected stored question: %+v", questionMessage)
	}

	answerMessage := storedMessages[1]
	if answerMessage.ID != response.MessageID {
		t.Errorf("expected the response message id %q to identify the answer, got %q", response.MessageID, answerMessage.ID)
	}
	if answerMessage.Role != "assistant" || answerMessage.Content != "it stores a question and an answer" {
		t.Errorf("unexpected stored answer: %+v", answerMessage)
	}
	if answerMessage.Reasoning != "history belongs to the backend" {
		t.Errorf("expected the trimmed reasoning to be stored, got %q", answerMessage.Reasoning)
	}
	if answerMessage.ThinkingDurationMs != 1234 {
		t.Errorf("expected the client duration to be stored, got %d", answerMessage.ThinkingDurationMs)
	}
	if len(answerMessage.Sources) != 2 {
		t.Fatalf("expected the source without repository, path or type to be dropped, got %d sources", len(answerMessage.Sources))
	}

	firstSource := answerMessage.Sources[0]
	if firstSource.Type != "code" || firstSource.Repo != "everest" || firstSource.Path != "internal/api/handler.go" {
		t.Errorf("unexpected first source mapping: %+v", firstSource)
	}
	if firstSource.Label != "everest/internal/api/handler.go" {
		t.Errorf("expected the label to combine repository and path, got %q", firstSource.Label)
	}
	if firstSource.StartLine != 12 || firstSource.EndLine != 30 {
		t.Errorf("expected the cited line range to be stored, got %d-%d", firstSource.StartLine, firstSource.EndLine)
	}
	secondSource := answerMessage.Sources[1]
	if secondSource.Path != "aurora/worker.go" || secondSource.Label != "aurora/worker.go" {
		t.Errorf("expected the path-only source to keep its path as label, got %+v", secondSource)
	}

	readBack, err := service.GetMessages(context.Background(), transcriptUserID, transcriptSessionID)
	if err != nil {
		t.Fatalf("cannot read the transcript back: %v", err)
	}
	if len(readBack) != 2 {
		t.Fatalf("expected the transcript to be readable, got %d messages", len(readBack))
	}
	if len(readBack[1].Sources) != 2 || readBack[1].Sources[0].Path != "internal/api/handler.go" {
		t.Errorf("expected the citations to survive the read path, got %+v", readBack[1].Sources)
	}

	encodedAnswer, err := json.Marshal(readBack[1])
	if err != nil {
		t.Fatalf("cannot encode the assistant message: %v", err)
	}
	serializedAnswer := string(encodedAnswer)
	for _, fragment := range []string{`"repo":"everest"`, `"path":"internal/api/handler.go"`, `"start_line":12`, `"end_line":30`} {
		if !strings.Contains(serializedAnswer, fragment) {
			t.Errorf("expected the serialized citation to contain %s, got %s", fragment, serializedAnswer)
		}
	}
	if strings.Contains(serializedAnswer, `"start_line":0`) {
		t.Error("expected absent line numbers to stay optional in the serialized citation")
	}

	storedSession := repo.sessions[transcriptSessionID]
	if !storedSession.UpdatedAt.After(sessionUpdatedAt) {
		t.Error("expected the session activity timestamp to move forward")
	}
	if storedSession.Title != "existing session" || storedSession.Model != "deepseek-flash" {
		t.Errorf("expected the session metadata to stay intact, got %+v", storedSession)
	}
}

func TestSaveTranscript_RejectsSessionOwnedByAnotherUser(t *testing.T) {
	repo := newFakeCopilotRepository()
	service := newTranscriptService(repo, transcriptOtherUser, time.Now().UTC())

	_, err := service.SaveTranscript(context.Background(), transcriptUserID, validTranscriptRequest())
	if !errors.Is(err, sharedErrors.ErrForbidden) {
		t.Fatalf("expected a forbidden error for a foreign session, got %v", err)
	}
	if repo.saveCalls != 0 || repo.updateCall != 0 {
		t.Errorf("expected no writes for a foreign session, got %d saves and %d updates", repo.saveCalls, repo.updateCall)
	}
	if len(repo.messages[transcriptSessionID]) != 0 {
		t.Fatal("expected the foreign session to gain no messages")
	}
}

func TestSaveTranscript_RejectsInvalidRequests(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(request *domain.TranscriptRequest)
	}{
		{"empty session id", func(request *domain.TranscriptRequest) { request.SessionID = "" }},
		{"blank session id", func(request *domain.TranscriptRequest) { request.SessionID = "   " }},
		{"empty question", func(request *domain.TranscriptRequest) { request.Question = "" }},
		{"blank question", func(request *domain.TranscriptRequest) { request.Question = " \n " }},
		{"empty answer", func(request *domain.TranscriptRequest) { request.Answer = "" }},
		{"blank answer", func(request *domain.TranscriptRequest) { request.Answer = "\t" }},
		{"negative duration", func(request *domain.TranscriptRequest) { request.DurationMs = -1 }},
	}

	for _, testCase := range testCases {
		repo := newFakeCopilotRepository()
		service := newTranscriptService(repo, transcriptUserID, time.Now().UTC())
		request := validTranscriptRequest()
		testCase.mutate(&request)

		_, err := service.SaveTranscript(context.Background(), transcriptUserID, request)
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Errorf("%s: expected a bad request error, got %v", testCase.name, err)
		}
		if repo.saveCalls != 0 {
			t.Errorf("%s: expected no message writes, got %d", testCase.name, repo.saveCalls)
		}
	}
}

func TestSaveTranscript_ReturnsNotFoundForUnknownSession(t *testing.T) {
	repo := newFakeCopilotRepository()
	service := NewCopilotService(repo, nil, nil, nil)

	_, err := service.SaveTranscript(context.Background(), transcriptUserID, validTranscriptRequest())
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected a not found error for a missing session, got %v", err)
	}
	if repo.saveCalls != 0 {
		t.Errorf("expected no message writes for a missing session, got %d", repo.saveCalls)
	}
}

func TestSaveTranscript_WrapsRepositoryFailures(t *testing.T) {
	testCases := []struct {
		name    string
		prepare func(repo *fakeCopilotRepository)
	}{
		{"session update failure", func(repo *fakeCopilotRepository) { repo.updateErr = sharedErrors.ErrInternal }},
		{"message save failure", func(repo *fakeCopilotRepository) { repo.saveErr = sharedErrors.ErrInternal }},
	}

	for _, testCase := range testCases {
		repo := newFakeCopilotRepository()
		service := newTranscriptService(repo, transcriptUserID, time.Now().UTC())
		testCase.prepare(repo)

		_, err := service.SaveTranscript(context.Background(), transcriptUserID, validTranscriptRequest())
		if !errors.Is(err, sharedErrors.ErrInternal) {
			t.Errorf("%s: expected the repository failure to surface, got %v", testCase.name, err)
		}
		if !strings.Contains(err.Error(), "saveTranscript") {
			t.Errorf("%s: expected wrapped context on the failure, got %v", testCase.name, err)
		}
	}
}
