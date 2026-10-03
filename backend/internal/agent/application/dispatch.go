package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

type EventType string

const (
	EventAgentStart     EventType = "agent_start"
	EventRoundStart     EventType = "round_start"
	EventRoundEnd       EventType = "round_end"
	EventPhaseChange    EventType = "phase_change"
	EventReasoningDelta EventType = "reasoning_delta"
	EventTextDelta      EventType = "text_delta"
	EventToolCallStart  EventType = "tool_call_start"
	EventToolArgsDelta  EventType = "tool_args_delta"
	EventToolResult     EventType = "tool_result"
	EventSources        EventType = "sources"
	EventDone           EventType = "done"
	EventError          EventType = "error"
)

type Event struct {
	Type    EventType
	Payload any
}

type EventFunc func(event Event) error

type AgentStartPayload struct {
	Question string
}

type RoundStartPayload struct {
	Round     int
	MaxRounds int
}

type RoundEndPayload struct {
	Round       int
	ToolsCalled int
}

type PhaseChangePayload struct {
	Phase string
}

type ReasoningDeltaPayload struct {
	Text string
}

type TextDeltaPayload struct {
	Text string
}

type ToolCallStartPayload struct {
	Round  int
	CallID string
	Name   string
	Action string
}

type ToolArgsDeltaPayload struct {
	Round  int
	CallID string
	Delta  string
}

type ToolResultPayload struct {
	Round       int
	Name        string
	ChunksFound int
	Preview     string
	Failed      bool
	Error       string
}

type SourcesPayload struct {
	Items []domain.SourceReference
}

type DonePayload struct {
	Rounds              int
	IsTruncated         bool
	TotalToolsCalled    int
	TotalEvidenceChunks int
	ElapsedMs           int64
}

type ErrorPayload struct {
	Message string
}

const (
	textChunkSize   = 24
	toolResultLimit = 200
)

type toolExecution struct {
	call    domain.ToolCall
	content string
	sources []domain.SourceReference
	failed  bool
	err     error
}

type sourceCollector struct {
	items    []domain.SourceReference
	seenKeys map[string]bool
}

func newSourceCollector() *sourceCollector {
	return &sourceCollector{seenKeys: make(map[string]bool)}
}

func (c *sourceCollector) add(sources ...domain.SourceReference) {
	for _, source := range sources {
		key := source.Repo + "|" + source.File + "|" + source.Snippet
		if c.seenKeys[key] {
			continue
		}
		c.seenKeys[key] = true
		c.items = append(c.items, source)
	}
}

func normalizeEventFunc(onEvent EventFunc) EventFunc {
	if onEvent == nil {
		return func(Event) error { return nil }
	}
	return onEvent
}

func dispatchToolCalls(ctx context.Context, registry domain.ToolRegistry, calls []domain.ToolCall) []toolExecution {
	executions := make([]toolExecution, len(calls))
	switch len(calls) {
	case 0:
		return executions
	case 1:
		executions[0] = executeToolCall(ctx, registry, calls[0])
		return executions
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(calls))
	for index, call := range calls {
		go func(position int, toolCall domain.ToolCall) {
			defer waitGroup.Done()
			executions[position] = executeToolCall(ctx, registry, toolCall)
		}(index, call)
	}
	waitGroup.Wait()
	return executions
}

func executeToolCall(ctx context.Context, registry domain.ToolRegistry, call domain.ToolCall) toolExecution {
	result, err := registry.Execute(ctx, call.Function.Name, json.RawMessage(call.Function.Arguments))
	if err != nil {
		return toolExecution{
			call:    call,
			content: fmt.Sprintf("Tool '%s' failed: %v", call.Function.Name, err),
			failed:  true,
			err:     err,
		}
	}
	return toolExecution{call: call, content: result.Content, sources: result.Sources}
}

func (a *Agent) executeToolBatch(ctx context.Context, state *runState, round int, calls []domain.ToolCall) error {
	for _, call := range calls {
		state.executedTools = append(state.executedTools, call.Function.Name)
		payload := ToolCallStartPayload{
			Round:  round,
			CallID: call.ID,
			Name:   call.Function.Name,
			Action: formatToolAction(call),
		}
		if err := state.emit(Event{Type: EventToolCallStart, Payload: payload}); err != nil {
			return fmt.Errorf("emit tool call start: %w", err)
		}
	}

	for _, execution := range dispatchToolCalls(ctx, a.registry, calls) {
		state.sources.add(execution.sources...)

		preview := execution.content
		if len(preview) > toolResultLimit {
			preview = preview[:toolResultLimit] + truncationSuffix
		}

		payload := ToolResultPayload{
			Round:       round,
			Name:        execution.call.Function.Name,
			ChunksFound: len(execution.sources),
			Preview:     preview,
			Failed:      execution.failed,
		}
		if execution.err != nil {
			payload.Error = execution.err.Error()
		}
		if err := state.emit(Event{Type: EventToolResult, Payload: payload}); err != nil {
			return fmt.Errorf("emit tool result: %w", err)
		}

		state.messages = append(state.messages, domain.ChatMessage{
			Role:       toolRole,
			Content:    domain.TextMessage(execution.content),
			ToolCallID: execution.call.ID,
		})
	}
	return nil
}

func emitTextAsChunks(emit EventFunc, text string) error {
	characters := []rune(text)
	for start := 0; start < len(characters); start += textChunkSize {
		end := start + textChunkSize
		if end > len(characters) {
			end = len(characters)
		}
		payload := TextDeltaPayload{Text: string(characters[start:end])}
		if err := emit(Event{Type: EventTextDelta, Payload: payload}); err != nil {
			return fmt.Errorf("emit text delta: %w", err)
		}
	}
	return nil
}

func emitSources(emit EventFunc, sources []domain.SourceReference) error {
	items := make([]domain.SourceReference, len(sources))
	copy(items, sources)
	if err := emit(Event{Type: EventSources, Payload: SourcesPayload{Items: items}}); err != nil {
		return fmt.Errorf("emit sources: %w", err)
	}
	return nil
}

func (s *runState) finishDirect(round int, text string, streamedLive bool, startedAt time.Time) (*codeindexdomain.AgentAnswer, error) {
	if !streamedLive && strings.TrimSpace(text) != "" {
		if err := emitTextAsChunks(s.emit, text); err != nil {
			return nil, err
		}
	}
	return s.completeRun(round, false, text, startedAt)
}

func (s *runState) completeRun(rounds int, isTruncated bool, answer string, startedAt time.Time) (*codeindexdomain.AgentAnswer, error) {
	if err := emitSources(s.emit, s.sources.items); err != nil {
		return nil, err
	}

	payload := DonePayload{
		Rounds:              rounds,
		IsTruncated:         isTruncated,
		TotalToolsCalled:    len(s.executedTools),
		TotalEvidenceChunks: len(s.sources.items),
		ElapsedMs:           time.Since(startedAt).Milliseconds(),
	}
	if err := s.emit(Event{Type: EventDone, Payload: payload}); err != nil {
		return nil, fmt.Errorf("emit done: %w", err)
	}

	return &codeindexdomain.AgentAnswer{
		Answer:      answer,
		Rounds:      rounds,
		IsTruncated: isTruncated,
		Sources:     toAgentSources(s.sources.items),
	}, nil
}

func toAgentSources(sources []domain.SourceReference) []codeindexdomain.AgentSource {
	if len(sources) == 0 {
		return nil
	}
	agentSources := make([]codeindexdomain.AgentSource, len(sources))
	for index, source := range sources {
		agentSources[index] = codeindexdomain.AgentSource{
			Repo:      source.Repo,
			File:      source.File,
			Type:      source.Type,
			Snippet:   source.Snippet,
			StartLine: source.StartLine,
			EndLine:   source.EndLine,
		}
	}
	return agentSources
}
