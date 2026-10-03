package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lunar/backend/internal/copilot/application"
	copilotDomain "lunar/backend/internal/copilot/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	transcriptTestJWTSecret = "copilot-transcript-jwt-secret"
	transcriptTestUserID    = "copilot-user-a"
	transcriptTestOtherUser = "copilot-user-b"
	transcriptTestSessionID = "copilot-session-1"
)

type fakeTranscriptRepository struct {
	sessions map[string]*copilotDomain.ChatSession
	messages map[string][]copilotDomain.ChatMessage
}

func newFakeTranscriptRepository() *fakeTranscriptRepository {
	return &fakeTranscriptRepository{
		sessions: make(map[string]*copilotDomain.ChatSession),
		messages: make(map[string][]copilotDomain.ChatMessage),
	}
}

func (f *fakeTranscriptRepository) seedSession(session copilotDomain.ChatSession) {
	storedCopy := session
	f.sessions[session.ID] = &storedCopy
}

func (f *fakeTranscriptRepository) GetSessionsByUserID(ctx context.Context, userID string) ([]copilotDomain.ChatSession, error) {
	sessionList := make([]copilotDomain.ChatSession, 0)
	for _, session := range f.sessions {
		if session.UserID == userID {
			sessionList = append(sessionList, *session)
		}
	}
	return sessionList, nil
}

func (f *fakeTranscriptRepository) GetSessionByID(ctx context.Context, sessionID string) (*copilotDomain.ChatSession, error) {
	session, exists := f.sessions[sessionID]
	if !exists {
		return nil, sharedErrors.ErrNotFound
	}
	storedCopy := *session
	return &storedCopy, nil
}

func (f *fakeTranscriptRepository) CreateSession(ctx context.Context, session *copilotDomain.ChatSession) error {
	f.seedSession(*session)
	return nil
}

func (f *fakeTranscriptRepository) UpdateSession(ctx context.Context, session *copilotDomain.ChatSession) error {
	f.seedSession(*session)
	return nil
}

func (f *fakeTranscriptRepository) DeleteSession(ctx context.Context, userID string, sessionID string) error {
	delete(f.sessions, sessionID)
	return nil
}

func (f *fakeTranscriptRepository) GetMessagesByChatID(ctx context.Context, chatID string) ([]copilotDomain.ChatMessage, error) {
	stored := f.messages[chatID]
	messageCopy := make([]copilotDomain.ChatMessage, len(stored))
	copy(messageCopy, stored)
	return messageCopy, nil
}

func (f *fakeTranscriptRepository) SaveMessage(ctx context.Context, msg *copilotDomain.ChatMessage) error {
	storedCopy := *msg
	f.messages[msg.ChatID] = append(f.messages[msg.ChatID], storedCopy)
	return nil
}

func setupTranscriptHandler(t *testing.T) (http.Handler, *fakeTranscriptRepository) {
	t.Helper()

	repo := newFakeTranscriptRepository()
	now := time.Now().UTC()
	repo.seedSession(copilotDomain.ChatSession{
		ID:        transcriptTestSessionID,
		UserID:    transcriptTestUserID,
		Title:     "seeded session",
		Model:     "deepseek-flash",
		CreatedAt: now,
		UpdatedAt: now,
	})

	service := application.NewCopilotService(repo, nil, nil, nil)
	handler := NewCopilotHandler(service)

	authMiddleware := sharedAuth.RequireAuth(transcriptTestJWTSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /api/copilot/transcript", authMiddleware(http.HandlerFunc(handler.SaveTranscript)))
	mux.Handle("GET /api/copilot/sessions/{id}/messages", authMiddleware(http.HandlerFunc(handler.ListMessages)))

	return mux, repo
}

func authorizedTranscriptRequest(t *testing.T, method string, path string, body string, userID string) *http.Request {
	t.Helper()

	var requestBody *bytes.Buffer
	if body == "" {
		requestBody = bytes.NewBuffer(nil)
	} else {
		requestBody = bytes.NewBufferString(body)
	}

	request := httptest.NewRequest(method, path, requestBody)
	request.Header.Set("Content-Type", "application/json")
	token, err := sharedAuth.GenerateToken(userID, userID+"@example.com", transcriptTestJWTSecret, time.Hour)
	if err != nil {
		t.Fatalf("cannot generate token: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return request
}

func performTranscriptRequest(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func transcriptPayload() string {
	return `{"session_id":"` + transcriptTestSessionID + `",` +
		`"question":"where does the transcript land?",` +
		`"answer":"in copilot transport",` +
		`"reasoning":"the helper answered on the laptop",` +
		`"duration_ms":4200,` +
		`"sources":[` +
		`{"repo":"everest","file":"internal/copilot/transport/handler.go","type":"code","snippet":"func (h *CopilotHandler) SaveTranscript","start_line":40,"end_line":70},` +
		`{"path":"aurora/worker.go","type":"code"}` +
		`]}`
}

func TestCopilotHandler_SaveTranscriptPersistsCitationsAndReadsThemBack(t *testing.T) {
	handler, repo := setupTranscriptHandler(t)

	recorder := performTranscriptRequest(handler, authorizedTranscriptRequest(t, http.MethodPost, "/api/copilot/transcript", transcriptPayload(), transcriptTestUserID))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response copilotDomain.TranscriptResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode transcript response: %v", err)
	}
	if response.MessageID == "" {
		t.Fatal("expected the transcript response to carry a message id")
	}

	storedMessages := repo.messages[transcriptTestSessionID]
	if len(storedMessages) != 2 {
		t.Fatalf("expected a question and an answer to be stored, got %d messages", len(storedMessages))
	}
	if storedMessages[0].Role != "user" || storedMessages[0].Content != "where does the transcript land?" {
		t.Errorf("unexpected stored question: %+v", storedMessages[0])
	}
	if storedMessages[1].ID != response.MessageID {
		t.Errorf("expected the response message id to identify the answer, got %q for %q", response.MessageID, storedMessages[1].ID)
	}
	if storedMessages[1].Reasoning != "the helper answered on the laptop" || storedMessages[1].ThinkingDurationMs != 4200 {
		t.Errorf("unexpected stored reasoning or duration: %+v", storedMessages[1])
	}
	if len(storedMessages[1].Sources) != 2 {
		t.Fatalf("expected both client citations to be stored, got %d", len(storedMessages[1].Sources))
	}

	readRecorder := performTranscriptRequest(handler, authorizedTranscriptRequest(t, http.MethodGet, "/api/copilot/sessions/"+transcriptTestSessionID+"/messages", "", transcriptTestUserID))
	if readRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 on read back, got %d: %s", readRecorder.Code, readRecorder.Body.String())
	}
	readBody := readRecorder.Body.String()

	var readMessages []copilotDomain.ChatMessage
	if err := json.NewDecoder(strings.NewReader(readBody)).Decode(&readMessages); err != nil {
		t.Fatalf("cannot decode read back messages: %v", err)
	}
	if len(readMessages) != 2 {
		t.Fatalf("expected the transcript to be readable back, got %d messages", len(readMessages))
	}
	if readMessages[1].Content != "in copilot transport" {
		t.Errorf("expected the stored answer on read back, got %q", readMessages[1].Content)
	}
	if len(readMessages[1].Sources) != 2 {
		t.Fatalf("expected the citations to survive the read path, got %d", len(readMessages[1].Sources))
	}
	firstSource := readMessages[1].Sources[0]
	if firstSource.Repo != "everest" || firstSource.Path != "internal/copilot/transport/handler.go" {
		t.Errorf("unexpected citation location: %+v", firstSource)
	}
	if firstSource.Label != "everest/internal/copilot/transport/handler.go" {
		t.Errorf("unexpected citation label: %q", firstSource.Label)
	}
	if firstSource.StartLine != 40 || firstSource.EndLine != 70 {
		t.Errorf("unexpected citation line range: %d-%d", firstSource.StartLine, firstSource.EndLine)
	}
	secondSource := readMessages[1].Sources[1]
	if secondSource.Path != "aurora/worker.go" || secondSource.Label != "aurora/worker.go" {
		t.Errorf("unexpected path-only citation: %+v", secondSource)
	}
	for _, expectedFragment := range []string{`"repo":"everest"`, `"path":"internal/copilot/transport/handler.go"`, `"start_line":40`, `"end_line":70`} {
		if !strings.Contains(readBody, expectedFragment) {
			t.Errorf("expected the serialized citations to contain %s, got %s", expectedFragment, readBody)
		}
	}
}

func TestCopilotHandler_SaveTranscriptRejectsUnauthenticatedRequests(t *testing.T) {
	handler, repo := setupTranscriptHandler(t)

	request := httptest.NewRequest(http.MethodPost, "/api/copilot/transcript", bytes.NewBufferString(transcriptPayload()))
	request.Header.Set("Content-Type", "application/json")
	recorder := performTranscriptRequest(handler, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(repo.messages[transcriptTestSessionID]) != 0 {
		t.Fatal("expected an unauthenticated request to persist nothing")
	}
}

func TestCopilotHandler_SaveTranscriptRejectsSessionsOfOtherUsers(t *testing.T) {
	handler, repo := setupTranscriptHandler(t)

	recorder := performTranscriptRequest(handler, authorizedTranscriptRequest(t, http.MethodPost, "/api/copilot/transcript", transcriptPayload(), transcriptTestOtherUser))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(repo.messages[transcriptTestSessionID]) != 0 {
		t.Fatal("expected a foreign session to gain no messages")
	}
}

func TestCopilotHandler_SaveTranscriptRejectsUnknownSessions(t *testing.T) {
	handler, _ := setupTranscriptHandler(t)

	body := strings.Replace(transcriptPayload(), transcriptTestSessionID, "session-does-not-exist", 1)
	recorder := performTranscriptRequest(handler, authorizedTranscriptRequest(t, http.MethodPost, "/api/copilot/transcript", body, transcriptTestUserID))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCopilotHandler_SaveTranscriptRejectsInvalidPayloads(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{"missing session id", `{"question":"q","answer":"a"}`},
		{"empty question", `{"session_id":"` + transcriptTestSessionID + `","question":"  ","answer":"a"}`},
		{"empty answer", `{"session_id":"` + transcriptTestSessionID + `","question":"q","answer":""}`},
		{"malformed json", `{"session_id":`},
		{"unknown field", `{"session_id":"` + transcriptTestSessionID + `","question":"q","answer":"a","unexpected":true}`},
		{"oversized body", `{"session_id":"` + transcriptTestSessionID + `","question":"q","answer":"` + strings.Repeat("a", 1048577) + `"}`},
	}

	for _, testCase := range testCases {
		handler, repo := setupTranscriptHandler(t)

		recorder := performTranscriptRequest(handler, authorizedTranscriptRequest(t, http.MethodPost, "/api/copilot/transcript", testCase.body, transcriptTestUserID))

		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: expected status 400, got %d: %s", testCase.name, recorder.Code, recorder.Body.String())
			continue
		}
		if len(repo.messages[transcriptTestSessionID]) != 0 {
			t.Errorf("%s: expected an invalid payload to persist nothing", testCase.name)
		}
	}
}
