package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const DefaultMaxRounds = 8

type reasoningStreamer interface {
	StreamAnswerWithReasoning(
		ctx context.Context,
		messages []domain.ChatMessage,
		model string,
		onReasoning func(chunk string) error,
		onChunk func(chunk string) error,
	) error
}

type Agent struct {
	aiClient       domain.AIClient
	registry       domain.ToolRegistry
	MaxRounds      int
	ToolChoiceFunc func(question string, round int, executedTools []string) (string, bool)
}

func NewAgent(aiClient domain.AIClient, registry domain.ToolRegistry) *Agent {
	return &Agent{aiClient: aiClient, registry: registry}
}

func (a *Agent) maxRounds() int {
	if a.MaxRounds > 0 {
		return a.MaxRounds
	}
	return DefaultMaxRounds
}

type phase1Streamed struct {
	reasoning bool
	content   bool
}

type roundOutcome int

const (
	roundCompleted roundOutcome = iota
	roundDirectAnswer
	roundNeedsSynthesis
)

type runState struct {
	emit          EventFunc
	messages      []domain.ChatMessage
	tools         []domain.ToolDefinition
	model         string
	sources       *sourceCollector
	executedTools []string
}

func (a *Agent) Run(
	ctx context.Context,
	question codeindexdomain.AgentQuestion,
	onEvent EventFunc,
) (*codeindexdomain.AgentAnswer, error) {
	startedAt := time.Now()
	state := &runState{
		emit:     normalizeEventFunc(onEvent),
		messages: buildInitialMessages(question),
		tools:    a.registry.ScopedDefinitions(question.Domains),
		model:    question.Model,
		sources:  newSourceCollector(),
	}

	if err := state.emit(Event{Type: EventAgentStart, Payload: AgentStartPayload{Question: question.Question}}); err != nil {
		return nil, fmt.Errorf("emit agent start: %w", err)
	}

	maxRounds := a.maxRounds()
	var isTruncated bool

	for round := 1; round <= maxRounds; round++ {
		roundStart := RoundStartPayload{Round: round, MaxRounds: maxRounds}
		if err := state.emit(Event{Type: EventRoundStart, Payload: roundStart}); err != nil {
			return nil, fmt.Errorf("emit round start: %w", err)
		}

		outcome, answer, err := a.runRound(ctx, state, round, question.Question, startedAt)
		if err != nil {
			return nil, a.fail(state, round, err)
		}
		switch outcome {
		case roundDirectAnswer:
			return answer, nil
		case roundNeedsSynthesis:
			return state.synthesize(ctx, a.aiClient, round, false, startedAt)
		}
		if round == maxRounds {
			isTruncated = true
		}
	}

	return state.synthesize(ctx, a.aiClient, maxRounds, isTruncated, startedAt)
}

func (a *Agent) fail(state *runState, round int, err error) error {
	loopError := fmt.Errorf("agent loop round %d: %w", round, err)
	payload := ErrorPayload{Message: loopError.Error()}
	if emitErr := state.emit(Event{Type: EventError, Payload: payload}); emitErr != nil {
		return errors.Join(loopError, fmt.Errorf("emit error event: %w", emitErr))
	}
	return loopError
}

func (a *Agent) runRound(
	ctx context.Context,
	state *runState,
	round int,
	question string,
	startedAt time.Time,
) (roundOutcome, *codeindexdomain.AgentAnswer, error) {
	toolChoice, isRequired := a.determineToolChoice(question, round, state.executedTools, len(state.tools) > 0)

	response, streamed, err := a.runToolCallRound(ctx, state, round, toolChoice)
	if err != nil {
		return roundCompleted, nil, err
	}

	if isRequired && (response.FinishReason == "stop" || len(response.ToolCalls) == 0) {
		retryResponse, retryStreamed, retryErr := a.retryRequiredRound(ctx, state, round, toolChoice)
		if retryErr != nil {
			return roundCompleted, nil, retryErr
		}
		streamed.reasoning = streamed.reasoning || retryStreamed.reasoning
		streamed.content = streamed.content || retryStreamed.content

		if retryResponse.FinishReason == "stop" || len(retryResponse.ToolCalls) == 0 {
			text := firstNonEmpty(retryResponse.Content.Text(), response.Content.Text())
			if strings.TrimSpace(text) == "" {
				return roundCompleted, nil, fmt.Errorf("required tool evidence was not provided by the model")
			}
			answer, finishErr := state.finishDirect(round, text, streamed.content, startedAt)
			if finishErr != nil {
				return roundCompleted, nil, finishErr
			}
			return roundDirectAnswer, answer, nil
		}
		response = retryResponse
	}

	if response.FinishReason == "stop" || len(response.ToolCalls) == 0 {
		text := response.Content.Text()
		if streamed.content {
			answer, finishErr := state.finishDirect(round, text, true, startedAt)
			return roundDirectAnswer, answer, finishErr
		}
		if strings.TrimSpace(text) == "" {
			return roundNeedsSynthesis, nil, nil
		}
		answer, finishErr := state.finishDirect(round, text, false, startedAt)
		return roundDirectAnswer, answer, finishErr
	}

	if response.ReasoningContent != "" && !streamed.reasoning {
		payload := ReasoningDeltaPayload{Text: response.ReasoningContent}
		if err := state.emit(Event{Type: EventReasoningDelta, Payload: payload}); err != nil {
			return roundCompleted, nil, fmt.Errorf("emit reasoning delta: %w", err)
		}
	}

	reasoning := response.ReasoningContent
	if reasoning == "" {
		reasoning = reasoningFallbackText
	}
	state.messages = append(state.messages, domain.ChatMessage{
		Role:             agentRole,
		Content:          response.Content,
		ReasoningContent: reasoning,
		ToolCalls:        response.ToolCalls,
	})

	if err := a.executeToolBatch(ctx, state, round, response.ToolCalls); err != nil {
		return roundCompleted, nil, err
	}

	roundEnd := RoundEndPayload{Round: round, ToolsCalled: len(response.ToolCalls)}
	if err := state.emit(Event{Type: EventRoundEnd, Payload: roundEnd}); err != nil {
		return roundCompleted, nil, fmt.Errorf("emit round end: %w", err)
	}
	return roundCompleted, nil, nil
}

func (a *Agent) retryRequiredRound(
	ctx context.Context,
	state *runState,
	round int,
	toolChoice string,
) (*domain.ToolCallResponse, phase1Streamed, error) {
	originalMessages := state.messages
	retryMessages := append(originalMessages, domain.ChatMessage{Role: userRole, Content: domain.TextMessage(requiredEvidenceReminderText)})
	state.messages = retryMessages

	response, streamed, err := a.runToolCallRound(ctx, state, round, toolChoice)
	if err != nil {
		state.messages = originalMessages
		return nil, streamed, fmt.Errorf("retry required tool round: %w", err)
	}
	if response.FinishReason == "stop" || len(response.ToolCalls) == 0 {
		state.messages = originalMessages
	}
	return response, streamed, nil
}

func (a *Agent) runToolCallRound(
	ctx context.Context,
	state *runState,
	round int,
	toolChoice string,
) (*domain.ToolCallResponse, phase1Streamed, error) {
	streamer, ok := a.aiClient.(domain.StreamToolCaller)
	if !ok {
		response, err := a.aiClient.RunToolCall(ctx, state.messages, state.tools, state.model, toolChoice)
		if err != nil {
			return nil, phase1Streamed{}, fmt.Errorf("run tool call: %w", err)
		}
		return response, phase1Streamed{}, nil
	}

	var streamed phase1Streamed
	filter := newContentFilter()
	callbacks := domain.ToolCallStreamCallbacks{
		OnReasoningDelta: func(delta string) error {
			streamed.reasoning = true
			payload := ReasoningDeltaPayload{Text: delta}
			return state.emit(Event{Type: EventReasoningDelta, Payload: payload})
		},
		OnContentDelta: func(delta string) error {
			visible := filter.accept(delta)
			if visible == "" {
				return nil
			}
			streamed.content = true
			return state.emit(Event{Type: EventTextDelta, Payload: TextDeltaPayload{Text: visible}})
		},
		OnToolCallStart: func(callID, name string) error {
			announced := domain.ToolCall{ID: callID, Type: "function", Function: domain.ToolCallFunction{Name: name}}
			payload := ToolCallStartPayload{Round: round, CallID: callID, Name: name, Action: formatToolAction(announced)}
			return state.emit(Event{Type: EventToolCallStart, Payload: payload})
		},
		OnToolCallArgsDelta: func(callID, delta string) error {
			payload := ToolArgsDeltaPayload{Round: round, CallID: callID, Delta: delta}
			return state.emit(Event{Type: EventToolArgsDelta, Payload: payload})
		},
	}

	response, err := streamer.RunToolCallStream(ctx, state.messages, state.tools, state.model, toolChoice, callbacks)
	if err != nil {
		return nil, streamed, fmt.Errorf("run tool call stream: %w", err)
	}
	return response, streamed, nil
}

func firstNonEmpty(primary string, secondary string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return secondary
}

func streamAnswer(
	ctx context.Context,
	client domain.AIClient,
	messages []domain.ChatMessage,
	model string,
	onReasoning func(chunk string) error,
	onChunk func(chunk string) error,
) error {
	if streamer, ok := client.(reasoningStreamer); ok {
		if err := streamer.StreamAnswerWithReasoning(ctx, messages, model, onReasoning, onChunk); err != nil {
			return fmt.Errorf("stream answer with reasoning: %w", err)
		}
		return nil
	}
	if err := client.StreamAnswer(ctx, messages, model, onChunk); err != nil {
		return fmt.Errorf("stream answer: %w", err)
	}
	return nil
}
