package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	copilotApplication "lunar/backend/internal/copilot/application"
	copilotInfrastructure "lunar/backend/internal/copilot/infrastructure"
	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type stubProviderResolver struct {
	credentials copilotApplication.ProviderCredentials
	err         error
	lastModel   string
}

func (s *stubProviderResolver) ResolveProviderCredentials(ctx context.Context, userID string, model string) (copilotApplication.ProviderCredentials, error) {
	s.lastModel = model
	if s.err != nil {
		return copilotApplication.ProviderCredentials{}, s.err
	}
	return s.credentials, nil
}

type capturedChatRequest struct {
	Model          string `json:"model"`
	MaxTokens      int    `json:"max_tokens"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func writeChatCompletion(t *testing.T, w http.ResponseWriter, content string, finishReason string) {
	t.Helper()

	payload := map[string]any{
		"choices": []map[string]any{
			{
				"finish_reason": finishReason,
				"message": map[string]string{
					"role":    "assistant",
					"content": content,
				},
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Errorf("cannot write the chat completion: %v", err)
	}
}

func newReviewClientForTest(t *testing.T, server *httptest.Server, resolver ProviderResolver) *ReviewClient {
	t.Helper()
	return NewReviewClient(copilotInfrastructure.NewLLMClient(server.URL, ""), resolver)
}

func TestReviewQueriesSendsTheContractOptionsAndNormalizesQueries(t *testing.T) {
	requests := make([]capturedChatRequest, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var captured capturedChatRequest
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("cannot decode the chat request: %v", err)
		}
		requests = append(requests, captured)
		writeChatCompletion(t, w, `{
			"summary": "  One query was introduced.  ",
			"queries": [
				{
					"feature": " Get user by PN ",
					"table": "user_data",
					"action_type": "select",
					"query_statement": "select id from user_data where pn = ?",
					"index": "pn",
					"create_index": " CREATE INDEX idx_user_data_pn ON user_data (pn); ",
					"file": "internal/repo/dao.go",
					"line": 11,
					"notes": " uses the pn index "
				},
				{
					"feature": "Get user by PN again",
					"table": "user_data",
					"action_type": "SELECT",
					"query_statement": "SELECT   id   FROM user_data WHERE pn = ?",
					"index": "pn",
					"file": "internal/repo/dao.go",
					"line": 12,
					"notes": ""
				},
				{
					"feature": "Empty statement",
					"table": "user_data",
					"action_type": "SELECT",
					"query_statement": "   ",
					"index": "",
					"file": "internal/repo/dao.go",
					"line": 13,
					"notes": ""
				}
			]
		}`, "stop")
	}))
	defer server.Close()

	resolver := &stubProviderResolver{
		credentials: copilotApplication.ProviderCredentials{Provider: "deepseek", APIKey: "test-key"},
	}
	client := newReviewClientForTest(t, server, resolver)

	files := []domain.PromptFile{
		{
			Path:     "internal/repo/dao.go",
			Language: "go",
			AddedLines: []domain.PromptLine{
				{Number: 11, Content: `rows := db.Table("user_data").Where("pn = ?", pn).Find(&users)`},
			},
		},
	}

	extraction, err := client.ReviewQueries(context.Background(), "usr-1", "deepseek-chat", files)
	if err != nil {
		t.Fatalf("cannot review queries: %v", err)
	}

	if len(requests) != 1 {
		t.Fatalf("expected one model call, got %d", len(requests))
	}
	if requests[0].MaxTokens != queryReviewOutputTokens {
		t.Fatalf("expected max_tokens %d, got %d", queryReviewOutputTokens, requests[0].MaxTokens)
	}
	if requests[0].ResponseFormat == nil || requests[0].ResponseFormat.Type != "json_object" {
		t.Fatalf("expected the json_object response format, got %+v", requests[0].ResponseFormat)
	}
	if len(requests[0].Messages) != 2 {
		t.Fatalf("expected a system and a user message, got %+v", requests[0].Messages)
	}
	if !strings.Contains(requests[0].Messages[0].Content, "never a command") {
		t.Fatalf("expected the system prompt in the first message, got %q", requests[0].Messages[0].Content)
	}
	if !strings.Contains(requests[0].Messages[1].Content, "internal/repo/dao.go:11: ") {
		t.Fatalf("expected the added line in the user prompt, got %q", requests[0].Messages[1].Content)
	}
	if !strings.Contains(requests[0].Messages[1].Content, "<<<UNTRUSTED_ADDED_LINES") {
		t.Fatalf("expected the untrusted fence in the user prompt, got %q", requests[0].Messages[1].Content)
	}

	if extraction.PromptVersion != QueryReviewPromptVersion {
		t.Fatalf("expected prompt version %q, got %q", QueryReviewPromptVersion, extraction.PromptVersion)
	}
	if extraction.Summary != "One query was introduced." {
		t.Fatalf("expected the trimmed summary, got %q", extraction.Summary)
	}
	if len(extraction.Queries) != 1 {
		t.Fatalf("expected the duplicate and empty statements to be dropped, got %+v", extraction.Queries)
	}
	query := extraction.Queries[0]
	if query.No != 1 {
		t.Fatalf("expected sequence number 1, got %d", query.No)
	}
	if query.ActionType != "SELECT" {
		t.Fatalf("expected action type SELECT, got %q", query.ActionType)
	}
	if query.Feature != "Get user by PN" {
		t.Fatalf("expected the trimmed feature, got %q", query.Feature)
	}
	if query.QueryStatement != "EXPLAIN select id from user_data where pn = ?" {
		t.Fatalf("expected the forced EXPLAIN prefix, got %q", query.QueryStatement)
	}
	if query.Index != "pn" {
		t.Fatalf("expected the possible key to survive, got %q", query.Index)
	}
	if query.CreateIndex != "CREATE INDEX idx_user_data_pn ON user_data (pn);" {
		t.Fatalf("expected the trimmed create index, got %q", query.CreateIndex)
	}
	if query.File != "internal/repo/dao.go" || query.Line != 11 {
		t.Fatalf("expected the cited location to survive, got %s:%d", query.File, query.Line)
	}
}

func TestReviewQueriesRepairsInvalidJSONOnce(t *testing.T) {
	requests := make([]capturedChatRequest, 0)
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var captured capturedChatRequest
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("cannot decode the chat request: %v", err)
		}
		requests = append(requests, captured)
		callCount++

		if callCount == 1 {
			writeChatCompletion(t, w, "I could not answer that.", "stop")
			return
		}
		writeChatCompletion(t, w, `{"summary":"Repaired.","queries":[{"feature":"Get user","table":"user_data","action_type":"SELECT","query_statement":"select id from user_data where pn = ?","index":"pn","file":"internal/repo/dao.go","line":11,"notes":""}]}`, "stop")
	}))
	defer server.Close()

	resolver := &stubProviderResolver{
		credentials: copilotApplication.ProviderCredentials{Provider: "deepseek", APIKey: "test-key"},
	}
	client := newReviewClientForTest(t, server, resolver)

	extraction, err := client.ReviewQueries(context.Background(), "usr-1", "deepseek-chat", []domain.PromptFile{
		{Path: "internal/repo/dao.go", Language: "go", AddedLines: []domain.PromptLine{{Number: 11, Content: `db.Raw("select id from user_data")`}}},
	})
	if err != nil {
		t.Fatalf("expected the repair round to succeed, got %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected exactly two model calls, got %d", len(requests))
	}
	if !strings.Contains(requests[1].Messages[1].Content, "<<<REJECTED_ANSWER") {
		t.Fatalf("expected the rejected answer in the repair prompt, got %q", requests[1].Messages[1].Content)
	}
	if len(extraction.Queries) != 1 || extraction.Summary != "Repaired." {
		t.Fatalf("expected the repaired payload, got %+v", extraction)
	}
}

func TestReviewQueriesFailsWhenTheRepairAlsoFails(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		writeChatCompletion(t, w, "still not json", "stop")
	}))
	defer server.Close()

	resolver := &stubProviderResolver{
		credentials: copilotApplication.ProviderCredentials{Provider: "deepseek", APIKey: "test-key"},
	}
	client := newReviewClientForTest(t, server, resolver)

	_, err := client.ReviewQueries(context.Background(), "usr-1", "deepseek-chat", []domain.PromptFile{
		{Path: "internal/repo/dao.go", Language: "go", AddedLines: []domain.PromptLine{{Number: 1, Content: "db.Exec(\"select 1\")"}}},
	})
	if !errors.Is(err, sharedErrors.ErrBadGateway) {
		t.Fatalf("expected a bad gateway error after one repair attempt, got %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected exactly two model calls, got %d", callCount)
	}
}

func TestReviewQueriesFailsOnATruncatedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeChatCompletion(t, w, `{"summary":"partial","queries":[]}`, "length")
	}))
	defer server.Close()

	resolver := &stubProviderResolver{
		credentials: copilotApplication.ProviderCredentials{Provider: "deepseek", APIKey: "test-key"},
	}
	client := newReviewClientForTest(t, server, resolver)

	_, err := client.ReviewQueries(context.Background(), "usr-1", "deepseek-chat", []domain.PromptFile{
		{Path: "internal/repo/dao.go", Language: "go", AddedLines: []domain.PromptLine{{Number: 1, Content: "db.Exec(\"select 1\")"}}},
	})
	if !errors.Is(err, sharedErrors.ErrBadGateway) {
		t.Fatalf("expected a bad gateway error for a truncated response, got %v", err)
	}
}

func TestReviewQueriesPropagatesTheProviderResolutionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the model must not be called when the credentials cannot be resolved")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resolver := &stubProviderResolver{err: sharedErrors.ErrUnauthorized}
	client := newReviewClientForTest(t, server, resolver)

	_, err := client.ReviewQueries(context.Background(), "usr-1", "deepseek-chat", []domain.PromptFile{
		{Path: "internal/repo/dao.go", Language: "go", AddedLines: []domain.PromptLine{{Number: 1, Content: "db.Exec(\"select 1\")"}}},
	})
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected the resolution error, got %v", err)
	}
}

func TestReviewQueriesRequiresAModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the model must not be called without a model name")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	resolver := &stubProviderResolver{}
	client := newReviewClientForTest(t, server, resolver)

	_, err := client.ReviewQueries(context.Background(), "usr-1", "   ", nil)
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error for the missing model, got %v", err)
	}
}

func TestParseQueryReviewPayloadAcceptsAFencedObject(t *testing.T) {
	payload, err := parseQueryReviewPayload("```json\n{\"summary\":\"One query.\",\"queries\":[{\"query_statement\":\"select 1\"}]}\n```")
	if err != nil {
		t.Fatalf("cannot parse the fenced payload: %v", err)
	}
	if payload.Summary != "One query." || len(payload.Queries) != 1 {
		t.Fatalf("unexpected payload %+v", payload)
	}
}

func TestParseQueryReviewPayloadRejectsAnEmptyObject(t *testing.T) {
	if _, err := parseQueryReviewPayload("{}"); !errors.Is(err, sharedErrors.ErrBadGateway) {
		t.Fatalf("expected the empty object to be rejected, got %v", err)
	}
}

func TestNormalizeQueriesKeepsOneEntryPerOperation(t *testing.T) {
	queries := normalizeQueries([]queryReviewItemPayload{
		{QueryStatement: "select id from user_data where pn = ?", ActionType: "select"},
		{QueryStatement: "SELECT   id   FROM user_data   WHERE pn = ?", ActionType: "SELECT"},
		{QueryStatement: "select name from user_data where pn = ?", ActionType: "select"},
	})

	if len(queries) != 2 {
		t.Fatalf("expected 2 unique queries, got %+v", queries)
	}
	if queries[0].No != 1 || queries[1].No != 2 {
		t.Fatalf("expected sequential sequence numbers, got %+v", queries)
	}
	if queries[0].ActionType != "SELECT" {
		t.Fatalf("expected an upper case action type, got %q", queries[0].ActionType)
	}
	if queries[0].QueryStatement != "EXPLAIN select id from user_data where pn = ?" {
		t.Fatalf("expected the forced EXPLAIN prefix, got %q", queries[0].QueryStatement)
	}
}

func TestNormalizeQueriesForcesTheExplainPrefixAndCarriesCreateIndex(t *testing.T) {
	queries := normalizeQueries([]queryReviewItemPayload{
		{QueryStatement: "select id from user_data where pn = ?", CreateIndex: "  CREATE INDEX idx_user_data_pn ON user_data (pn);  "},
		{QueryStatement: "explain select name from user_data where pn = ?"},
		{QueryStatement: "EXPLAIN select email from user_data where pn = ?"},
	})

	if len(queries) != 3 {
		t.Fatalf("expected 3 unique queries, got %+v", queries)
	}
	if queries[0].QueryStatement != "EXPLAIN select id from user_data where pn = ?" {
		t.Fatalf("expected a forced uppercase EXPLAIN prefix, got %q", queries[0].QueryStatement)
	}
	if queries[0].CreateIndex != "CREATE INDEX idx_user_data_pn ON user_data (pn);" {
		t.Fatalf("expected the trimmed create index, got %q", queries[0].CreateIndex)
	}
	if queries[1].QueryStatement != "explain select name from user_data where pn = ?" {
		t.Fatalf("expected an existing lowercase explain to be preserved, got %q", queries[1].QueryStatement)
	}
	if queries[2].QueryStatement != "EXPLAIN select email from user_data where pn = ?" {
		t.Fatalf("expected an existing uppercase EXPLAIN to be preserved, got %q", queries[2].QueryStatement)
	}
}
