package infrastructure

import (
	"testing"

	"github.com/google/uuid"

	"lunar/backend/internal/copilot/domain"
	sharedHttp "lunar/backend/internal/shared/http"
)

func TestChatCompletionOpenCodeGoSendsTheConfiguredSessionID(t *testing.T) {
	stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonStop, cannedChatAnswer, ""))
	client := NewLLMClient("", "")
	client.openCodeGoURL = stub.url()

	content, _, err := requestChatCompletion(t, client, domain.OpenCodeGoProvider, "opencode-go-deepseek-v4.1-flash", ChatOptions{SessionID: " conversation-42 "})
	if err != nil {
		t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
	}
	if content != cannedChatAnswer {
		t.Fatalf("content = %q, want %q", content, cannedChatAnswer)
	}

	request := requireSingleRecordedRequest(t, stub)
	if got := request.header.Get(openCodeSessionHeader); got != "conversation-42" {
		t.Errorf("%s = %q, want %q", openCodeSessionHeader, got, "conversation-42")
	}
	if got := request.header.Get("User-Agent"); got != sharedHttp.ClientUserAgent {
		t.Errorf("User-Agent = %q, want %q", got, sharedHttp.ClientUserAgent)
	}
	if got := request.body["model"]; got != "deepseek-v4.1-flash" {
		t.Errorf("model = %v, want %q", got, "deepseek-v4.1-flash")
	}
}

func TestChatCompletionOpenCodeGoGeneratesASessionIDWhenMissing(t *testing.T) {
	stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonStop, cannedChatAnswer, ""))
	client := NewLLMClient("", "")
	client.openCodeGoURL = stub.url()

	if _, _, err := requestChatCompletion(t, client, domain.OpenCodeGoProvider, "opencode-go-deepseek-v4.1-flash", ChatOptions{}); err != nil {
		t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
	}

	request := requireSingleRecordedRequest(t, stub)
	sessionID := request.header.Get(openCodeSessionHeader)
	if _, err := uuid.Parse(sessionID); err != nil {
		t.Errorf("%s = %q, want a UUID: %v", openCodeSessionHeader, sessionID, err)
	}
}

func TestChatCompletionOtherProvidersOmitTheSessionID(t *testing.T) {
	stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonStop, cannedChatAnswer, ""))
	client := NewLLMClient(stub.url(), "")

	if _, _, err := requestChatCompletion(t, client, deepseekProvider, chatModelName, ChatOptions{SessionID: "conversation-42"}); err != nil {
		t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
	}

	request := requireSingleRecordedRequest(t, stub)
	if got := request.header.Get(openCodeSessionHeader); got != "" {
		t.Errorf("%s = %q, want empty for a non opencode-go provider", openCodeSessionHeader, got)
	}
}
