package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"lunar/backend/internal/copilot/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	testAPIKey             = "stub-api-key"
	testUserPrompt         = "Summarize the workspace."
	testSystemPrompt       = "You are Lunar, a workspace copilot."
	chatModelName          = "DeepSeek Flash"
	reasoningModelName     = "DeepSeek-V4 Pro (Thinking)"
	mimoModelName          = "Xiaomi MiMo-V2.5 Pro"
	chatUpstreamModel      = "deepseek-chat"
	reasoningUpstreamModel = "deepseek-reasoner"
	mimoUpstreamModel      = "mimo-v2.5-pro"
	deepseekProvider       = "deepseek"
	mimoProvider           = "mimo"
	chatCompletionPath     = "/chat/completions"
	jsonContentType        = "application/json"
	finishReasonStop       = "stop"
	finishReasonLength     = "length"
	jsonObjectFormat       = "json_object"
	cannedChatAnswer       = "Canned chat answer."
	cannedJSONAnswer       = `{"answer":"canned"}`
)

type stubResponse struct {
	statusCode int
	body       string
}

type recordedChatRequest struct {
	method        string
	path          string
	authorization string
	contentType   string
	header        http.Header
	body          map[string]any
}

type chatStubServer struct {
	server         *httptest.Server
	mutex          sync.Mutex
	requests       []recordedChatRequest
	responses      []stubResponse
	handlerFailure error
}

func newChatStubServer(t *testing.T, responses ...stubResponse) *chatStubServer {
	t.Helper()
	stub := &chatStubServer{responses: responses}
	stub.server = httptest.NewServer(http.HandlerFunc(stub.handleChatCompletion))
	t.Cleanup(stub.server.Close)
	return stub
}

func (s *chatStubServer) url() string {
	return s.server.URL
}

func (s *chatStubServer) handleChatCompletion(writer http.ResponseWriter, request *http.Request) {
	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		s.recordHandlerFailure(readErr)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	var decodedBody map[string]any
	if decodeErr := json.Unmarshal(rawBody, &decodedBody); decodeErr != nil {
		s.recordHandlerFailure(decodeErr)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := s.appendRequest(recordedChatRequest{
		method:        request.Method,
		path:          request.URL.Path,
		authorization: request.Header.Get("Authorization"),
		contentType:   request.Header.Get("Content-Type"),
		header:        request.Header.Clone(),
		body:          decodedBody,
	})

	writer.Header().Set("Content-Type", jsonContentType)
	writer.WriteHeader(response.statusCode)
	if _, writeErr := writer.Write([]byte(response.body)); writeErr != nil {
		s.recordHandlerFailure(writeErr)
	}
}

func (s *chatStubServer) appendRequest(recorded recordedChatRequest) stubResponse {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	requestIndex := len(s.requests)
	s.requests = append(s.requests, recorded)
	if requestIndex < len(s.responses) {
		return s.responses[requestIndex]
	}
	return stubResponse{
		statusCode: http.StatusInternalServerError,
		body:       `{"error":{"message":"stub server received an unexpected request","type":"stub_error"}}`,
	}
}

func (s *chatStubServer) recordHandlerFailure(failure error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.handlerFailure = failure
}

func (s *chatStubServer) recordedRequests(t *testing.T) []recordedChatRequest {
	t.Helper()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.handlerFailure != nil {
		t.Fatalf("stub server failed to serve a request: %v", s.handlerFailure)
	}
	requests := make([]recordedChatRequest, len(s.requests))
	copy(requests, s.requests)
	return requests
}

func stubChatCompletionResponse(t *testing.T, finishReason string, content string, reasoningContent string) stubResponse {
	t.Helper()
	message := map[string]any{
		"role":    "assistant",
		"content": content,
	}
	if reasoningContent != "" {
		message["reasoning_content"] = reasoningContent
	}
	payload := map[string]any{
		"choices": []map[string]any{
			{
				"finish_reason": finishReason,
				"message":       message,
			},
		},
	}
	encoded, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		t.Fatalf("failed to encode the stub chat completion: %v", marshalErr)
	}
	return stubResponse{statusCode: http.StatusOK, body: string(encoded)}
}

func requestChatCompletion(t *testing.T, client *LLMClient, provider string, modelName string, options ChatOptions) (string, string, error) {
	t.Helper()
	return client.ChatCompletion(context.Background(), provider, testAPIKey, modelName, testUserPrompt, testSystemPrompt, false, domain.ReasoningEffortMedium, options)
}

func requireSingleRecordedRequest(t *testing.T, stub *chatStubServer) recordedChatRequest {
	t.Helper()
	requests := stub.recordedRequests(t)
	if len(requests) != 1 {
		t.Fatalf("recorded %d requests, want exactly 1", len(requests))
	}
	return requests[0]
}

func requireResponseFormat(t *testing.T, request recordedChatRequest) map[string]any {
	t.Helper()
	rawFormat, found := request.body["response_format"]
	if !found {
		t.Fatal("request body has no response_format field")
	}
	format, isObject := rawFormat.(map[string]any)
	if !isObject {
		t.Fatalf("response_format = %T, want a JSON object", rawFormat)
	}
	return format
}

func requireNoResponseFormat(t *testing.T, request recordedChatRequest) {
	t.Helper()
	if rawFormat, found := request.body["response_format"]; found {
		t.Errorf("response_format = %v, want the field to be absent", rawFormat)
	}
}

func TestChatCompletionSendsRequestedMaxOutputTokens(t *testing.T) {
	tests := []struct {
		name               string
		modelName          string
		upstreamModel      string
		maxOutputTokens    int
		expectsMaxTokens   bool
		expectsTemperature bool
	}{
		{
			name:               "chat model",
			modelName:          chatModelName,
			upstreamModel:      chatUpstreamModel,
			maxOutputTokens:    4096,
			expectsMaxTokens:   true,
			expectsTemperature: true,
		},
		{
			name:               "reasoning model",
			modelName:          reasoningModelName,
			upstreamModel:      reasoningUpstreamModel,
			maxOutputTokens:    2048,
			expectsMaxTokens:   true,
			expectsTemperature: false,
		},
		{
			name:               "chat model without a token budget",
			modelName:          chatModelName,
			upstreamModel:      chatUpstreamModel,
			maxOutputTokens:    0,
			expectsMaxTokens:   false,
			expectsTemperature: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonStop, cannedChatAnswer, ""))
			client := NewLLMClient(stub.url(), "")

			content, _, err := requestChatCompletion(t, client, deepseekProvider, testCase.modelName, ChatOptions{MaxOutputTokens: testCase.maxOutputTokens})
			if err != nil {
				t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
			}
			if content != cannedChatAnswer {
				t.Fatalf("content = %q, want %q", content, cannedChatAnswer)
			}

			request := requireSingleRecordedRequest(t, stub)
			if request.method != http.MethodPost {
				t.Errorf("method = %q, want %q", request.method, http.MethodPost)
			}
			if request.path != chatCompletionPath {
				t.Errorf("path = %q, want %q", request.path, chatCompletionPath)
			}
			if request.authorization != "Bearer "+testAPIKey {
				t.Errorf("authorization = %q, want %q", request.authorization, "Bearer "+testAPIKey)
			}
			if request.contentType != jsonContentType {
				t.Errorf("content type = %q, want %q", request.contentType, jsonContentType)
			}
			if request.body["model"] != testCase.upstreamModel {
				t.Errorf("model = %v, want %q", request.body["model"], testCase.upstreamModel)
			}

			rawMaxTokens, hasMaxTokens := request.body["max_tokens"]
			if hasMaxTokens != testCase.expectsMaxTokens {
				t.Errorf("max_tokens %v present = %t, want %t", rawMaxTokens, hasMaxTokens, testCase.expectsMaxTokens)
			}
			if testCase.expectsMaxTokens && rawMaxTokens != float64(testCase.maxOutputTokens) {
				t.Errorf("max_tokens = %v, want %d", rawMaxTokens, testCase.maxOutputTokens)
			}

			_, hasTemperature := request.body["temperature"]
			if hasTemperature != testCase.expectsTemperature {
				t.Errorf("temperature present = %t, want %t", hasTemperature, testCase.expectsTemperature)
			}
		})
	}
}

func TestChatCompletionSendsJSONResponseFormatOnlyWhenRequested(t *testing.T) {
	t.Run("json output enabled", func(t *testing.T) {
		stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonStop, cannedJSONAnswer, ""))
		client := NewLLMClient(stub.url(), "")

		content, _, err := requestChatCompletion(t, client, deepseekProvider, chatModelName, ChatOptions{JSONOutput: true})
		if err != nil {
			t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
		}
		if content != cannedJSONAnswer {
			t.Fatalf("content = %q, want %q", content, cannedJSONAnswer)
		}

		format := requireResponseFormat(t, requireSingleRecordedRequest(t, stub))
		if len(format) != 1 {
			t.Errorf("response_format = %v, want exactly one field", format)
		}
		if format["type"] != jsonObjectFormat {
			t.Errorf("response_format type = %v, want %q", format["type"], jsonObjectFormat)
		}
	})

	t.Run("json output disabled", func(t *testing.T) {
		stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonStop, cannedChatAnswer, ""))
		client := NewLLMClient(stub.url(), "")

		content, _, err := requestChatCompletion(t, client, deepseekProvider, chatModelName, ChatOptions{JSONOutput: false})
		if err != nil {
			t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
		}
		if content != cannedChatAnswer {
			t.Fatalf("content = %q, want %q", content, cannedChatAnswer)
		}

		requireNoResponseFormat(t, requireSingleRecordedRequest(t, stub))
	})
}

func TestChatCompletionOmitsJSONResponseFormatForUnlistedProvider(t *testing.T) {
	stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonStop, cannedChatAnswer, ""))
	client := NewLLMClient("", stub.url())

	content, _, err := requestChatCompletion(t, client, mimoProvider, mimoModelName, ChatOptions{JSONOutput: true})
	if err != nil {
		t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
	}
	if content != cannedChatAnswer {
		t.Fatalf("content = %q, want %q", content, cannedChatAnswer)
	}

	request := requireSingleRecordedRequest(t, stub)
	if request.body["model"] != mimoUpstreamModel {
		t.Errorf("model = %v, want %q", request.body["model"], mimoUpstreamModel)
	}
	requireNoResponseFormat(t, request)
}

func TestChatCompletionRetriesWithoutJSONResponseFormatAfterBadRequest(t *testing.T) {
	stub := newChatStubServer(t,
		stubResponse{
			statusCode: http.StatusBadRequest,
			body:       `{"error":{"message":"response_format is not supported by this model","type":"invalid_request_error"}}`,
		},
		stubChatCompletionResponse(t, finishReasonStop, cannedChatAnswer, ""),
	)
	client := NewLLMClient(stub.url(), "")

	content, _, err := requestChatCompletion(t, client, deepseekProvider, chatModelName, ChatOptions{JSONOutput: true})
	if err != nil {
		t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
	}
	if content != cannedChatAnswer {
		t.Fatalf("content = %q, want %q", content, cannedChatAnswer)
	}

	requests := stub.recordedRequests(t)
	if len(requests) != 2 {
		t.Fatalf("recorded %d requests, want 2", len(requests))
	}

	firstFormat := requireResponseFormat(t, requests[0])
	if firstFormat["type"] != jsonObjectFormat {
		t.Errorf("first response_format type = %v, want %q", firstFormat["type"], jsonObjectFormat)
	}

	if requests[1].body["model"] != chatUpstreamModel {
		t.Errorf("retried model = %v, want %q", requests[1].body["model"], chatUpstreamModel)
	}
	requireNoResponseFormat(t, requests[1])
}

func TestChatCompletionRejectsTruncatedResponseWhenFailOnTruncation(t *testing.T) {
	truncatedContent := "  Partial answer that hit the output limit  "
	trimmedContent := "Partial answer that hit the output limit"

	t.Run("fail on truncation enabled", func(t *testing.T) {
		stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonLength, truncatedContent, ""))
		client := NewLLMClient(stub.url(), "")

		content, reasoning, err := requestChatCompletion(t, client, deepseekProvider, chatModelName, ChatOptions{FailOnTruncation: true})
		if err == nil {
			t.Fatal("ChatCompletion returned no error for a truncated response with FailOnTruncation enabled")
		}
		if !errors.Is(err, sharedErrors.ErrBadGateway) {
			t.Errorf("error = %v, want an error wrapping %v", err, sharedErrors.ErrBadGateway)
		}
		if content != "" {
			t.Errorf("content = %q, want empty content when truncation fails the call", content)
		}
		if reasoning != "" {
			t.Errorf("reasoning = %q, want empty reasoning when truncation fails the call", reasoning)
		}
	})

	t.Run("fail on truncation disabled", func(t *testing.T) {
		stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonLength, truncatedContent, ""))
		client := NewLLMClient(stub.url(), "")

		content, reasoning, err := requestChatCompletion(t, client, deepseekProvider, chatModelName, ChatOptions{FailOnTruncation: false})
		if err != nil {
			t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
		}
		if content != trimmedContent {
			t.Errorf("content = %q, want %q", content, trimmedContent)
		}
		if reasoning != "" {
			t.Errorf("reasoning = %q, want empty reasoning without thinking support", reasoning)
		}
	})
}

func TestChatCompletionTrimsContentWhenFinishReasonIsStop(t *testing.T) {
	t.Run("chat model", func(t *testing.T) {
		stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonStop, "\n  Final chat answer.  \t\n", ""))
		client := NewLLMClient(stub.url(), "")

		content, reasoning, err := requestChatCompletion(t, client, deepseekProvider, chatModelName, ChatOptions{})
		if err != nil {
			t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
		}
		if content != "Final chat answer." {
			t.Errorf("content = %q, want %q", content, "Final chat answer.")
		}
		if reasoning != "" {
			t.Errorf("reasoning = %q, want empty reasoning without thinking support", reasoning)
		}
	})

	t.Run("reasoning model", func(t *testing.T) {
		stub := newChatStubServer(t, stubChatCompletionResponse(t, finishReasonStop, "  Final reasoned answer.  ", "  Step one, then step two.  "))
		client := NewLLMClient(stub.url(), "")

		content, reasoning, err := requestChatCompletion(t, client, deepseekProvider, reasoningModelName, ChatOptions{})
		if err != nil {
			t.Fatalf("ChatCompletion returned an unexpected error: %v", err)
		}
		if content != "Final reasoned answer." {
			t.Errorf("content = %q, want %q", content, "Final reasoned answer.")
		}
		if reasoning != "Step one, then step two." {
			t.Errorf("reasoning = %q, want %q", reasoning, "Step one, then step two.")
		}
	})
}
