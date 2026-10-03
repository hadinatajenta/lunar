package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	agentapplication "lunar/backend/internal/agent/application"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	sharedHttp "lunar/backend/internal/shared/http"
)

const (
	agentAskPath = "/agent/ask"

	agentRequestTimeout   = 5 * time.Minute
	maxQuestionCharacters = 8000
	maxDomains            = 8
)

type agentAskRequest struct {
	Question        string   `json:"question"`
	Root            string   `json:"root"`
	Domains         []string `json:"domains"`
	Provider        string   `json:"provider"`
	Model           string   `json:"model"`
	BackendURL      string   `json:"backend_url"`
	SessionToken    string   `json:"session_token"`
	ThinkingMode    bool     `json:"thinking_mode"`
	ReasoningEffort string   `json:"reasoning_effort"`
}

type agentSourcePayload struct {
	Repo      string `json:"repo"`
	File      string `json:"file"`
	Type      string `json:"type"`
	Snippet   string `json:"snippet"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

type agentEvent struct {
	Type    string               `json:"type"`
	Text    string               `json:"text,omitempty"`
	Tool    string               `json:"tool,omitempty"`
	Action  string               `json:"action,omitempty"`
	Round   int                  `json:"round,omitempty"`
	Sources []agentSourcePayload `json:"sources,omitempty"`
	Rounds  int                  `json:"rounds,omitempty"`
	Message string               `json:"message,omitempty"`
}

func (s *helperServer) handleAgentAsk(w http.ResponseWriter, r *http.Request) {
	if s.indexStore == nil {
		sharedHttp.WriteError(w, http.StatusServiceUnavailable, "the code index is not available")
		return
	}

	var request agentAskRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, "invalid agent request")
		return
	}

	question := strings.TrimSpace(request.Question)
	if question == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, "a question is required")
		return
	}
	if len(question) > maxQuestionCharacters {
		sharedHttp.WriteError(w, http.StatusBadRequest, "the question is too long")
		return
	}
	if len(request.Domains) > maxDomains {
		sharedHttp.WriteError(w, http.StatusBadRequest, "too many tool domains requested")
		return
	}

	backendURL := strings.TrimRight(strings.TrimSpace(request.BackendURL), "/")
	sessionToken := strings.TrimSpace(request.SessionToken)
	model := strings.TrimSpace(request.Model)
	if backendURL == "" || sessionToken == "" || model == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, "backend_url, session_token and model are required")
		return
	}

	credentials, err := fetchProviderCredentials(r.Context(), backendURL, sessionToken, model)
	if err != nil {
		s.logger.Warn("cannot fetch provider credentials", "error", err)
		sharedHttp.WriteError(w, http.StatusBadGateway, "cannot fetch the provider credentials from the backend")
		return
	}

	flusher, isFlushable := w.(http.Flusher)
	if !isFlushable {
		sharedHttp.WriteError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	defer func() {
		if _, writeError := fmt.Fprint(w, "data: [DONE]\n\n"); writeError != nil {
			return
		}
		flusher.Flush()
	}()

	writeEvent := func(event agentEvent) error {
		payload, marshalError := json.Marshal(event)
		if marshalError != nil {
			return marshalError
		}
		if _, writeError := fmt.Fprintf(w, "data: %s\n\n", payload); writeError != nil {
			return writeError
		}
		flusher.Flush()
		return nil
	}

	agentContext, cancel := context.WithTimeout(r.Context(), agentRequestTimeout)
	defer cancel()

	agent := s.buildAgent(request.Root, credentials)

	answer, runError := agent.Run(
		agentContext,
		codeindexdomain.AgentQuestion{
			Question: question,
			Model:    credentials.Model,
			Domains:  request.Domains,
		},
		func(event agentapplication.Event) error {
			translated, isTranslatable := translateAgentEvent(event)
			if !isTranslatable {
				return nil
			}
			return writeEvent(translated)
		},
	)

	if runError != nil {
		s.logger.Warn("agent run failed", "error", runError)
		_ = writeEvent(agentEvent{Type: "error", Message: runError.Error()})
		return
	}

	if answer != nil {
		sources := make([]agentSourcePayload, 0, len(answer.Sources))
		for _, source := range answer.Sources {
			sources = append(sources, agentSourcePayload{
				Repo:      source.Repo,
				File:      source.File,
				Type:      source.Type,
				Snippet:   source.Snippet,
				StartLine: source.StartLine,
				EndLine:   source.EndLine,
			})
		}
		_ = writeEvent(agentEvent{Type: "sources", Sources: sources})
		_ = writeEvent(agentEvent{Type: "done", Rounds: answer.Rounds})
	}
}

func translateAgentEvent(event agentapplication.Event) (agentEvent, bool) {
	switch event.Type {
	case agentapplication.EventReasoningDelta:
		payload, ok := event.Payload.(agentapplication.ReasoningDeltaPayload)
		if !ok {
			return agentEvent{}, false
		}
		return agentEvent{Type: "reasoning", Text: payload.Text}, true

	case agentapplication.EventTextDelta:
		payload, ok := event.Payload.(agentapplication.TextDeltaPayload)
		if !ok {
			return agentEvent{}, false
		}
		return agentEvent{Type: "text", Text: payload.Text}, true

	case agentapplication.EventToolCallStart:
		payload, ok := event.Payload.(agentapplication.ToolCallStartPayload)
		if !ok {
			return agentEvent{}, false
		}
		return agentEvent{Type: "tool_call", Tool: payload.Name, Action: payload.Action, Round: payload.Round}, true

	case agentapplication.EventToolResult:
		payload, ok := event.Payload.(agentapplication.ToolResultPayload)
		if !ok {
			return agentEvent{}, false
		}
		return agentEvent{Type: "tool_result", Tool: payload.Name, Round: payload.Round}, true

	case agentapplication.EventError:
		payload, ok := event.Payload.(agentapplication.ErrorPayload)
		if !ok {
			return agentEvent{}, false
		}
		return agentEvent{Type: "error", Message: payload.Message}, true

	default:
		return agentEvent{}, false
	}
}
