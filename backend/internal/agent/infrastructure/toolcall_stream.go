package infrastructure

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"lunar/backend/internal/agent/domain"
)

const (
	sseDataPrefix          = "data: "
	sseDoneMarker          = "[DONE]"
	defaultCallType        = "function"
	reasoningFallbackValue = "Evaluating tools for verification."
)

func sseData(line string) (string, bool) {
	if !strings.HasPrefix(line, sseDataPrefix) {
		return "", false
	}
	return strings.TrimPrefix(line, sseDataPrefix), true
}

type toolCallDeltaFragment struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type streamToolCallDelta struct {
	Content          string                  `json:"content"`
	ReasoningContent string                  `json:"reasoning_content"`
	ToolCalls        []toolCallDeltaFragment `json:"tool_calls"`
}

func (c *ChatClient) RunToolCallStream(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	modelOption string,
	toolChoice string,
	callbacks domain.ToolCallStreamCallbacks,
) (*domain.ToolCallResponse, error) {
	if strings.TrimSpace(c.cfg.APIKey) == "" {
		return nil, errors.New(missingAPIKeyMessage)
	}

	encoded, err := marshalRequestBody(c.buildToolCallRequestBody(messages, tools, modelOption, toolChoice, true))
	if err != nil {
		return nil, err
	}

	resp, err := c.sendJSON(ctx, c.cfg.BaseURL+chatCompletionsPath, encoded)
	if err != nil {
		return nil, fmt.Errorf("run tool call stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("run tool call stream: read error body: %w", readErr)
		}
		return nil, parseProviderError(c.providerName(), resp.StatusCode, body)
	}

	var (
		contentBuilder     strings.Builder
		reasoningBuilder   strings.Builder
		finishReason       string
		responseModel      string
		callsByIndex       = make(map[int]*domain.ToolCall)
		indexOrder         []int
		announcedCallIndex = make(map[int]bool)
	)

	handleFragment := func(fragment toolCallDeltaFragment) error {
		call, exists := callsByIndex[fragment.Index]
		if !exists {
			call = &domain.ToolCall{Type: fragment.Type}
			if call.Type == "" {
				call.Type = defaultCallType
			}
			callsByIndex[fragment.Index] = call
			indexOrder = append(indexOrder, fragment.Index)
		}
		if fragment.ID != "" && call.ID == "" {
			call.ID = fragment.ID
		}
		if fragment.Function.Name != "" && call.Function.Name == "" {
			call.Function.Name = fragment.Function.Name
		}
		if fragment.Function.Arguments != "" {
			call.Function.Arguments += fragment.Function.Arguments
		}

		switch {
		case !announcedCallIndex[fragment.Index] && call.Function.Name != "":
			announcedCallIndex[fragment.Index] = true
			if callbacks.OnToolCallStart != nil {
				if err := callbacks.OnToolCallStart(call.ID, call.Function.Name); err != nil {
					return err
				}
			}
			if call.Function.Arguments != "" && callbacks.OnToolCallArgsDelta != nil {
				if err := callbacks.OnToolCallArgsDelta(call.ID, call.Function.Arguments); err != nil {
					return err
				}
			}
		case announcedCallIndex[fragment.Index] && fragment.Function.Arguments != "":
			if callbacks.OnToolCallArgsDelta != nil {
				if err := callbacks.OnToolCallArgsDelta(call.ID, fragment.Function.Arguments); err != nil {
					return err
				}
			}
		}
		return nil
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 512*1024)

	for scanner.Scan() {
		data, isData := sseData(scanner.Text())
		if !isData {
			continue
		}
		if data == sseDoneMarker {
			break
		}

		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				FinishReason *string             `json:"finish_reason"`
				Delta        streamToolCallDelta `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Model != "" && responseModel == "" {
			responseModel = chunk.Model
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		if choice.Delta.ReasoningContent != "" {
			reasoningBuilder.WriteString(choice.Delta.ReasoningContent)
			if callbacks.OnReasoningDelta != nil {
				if err := callbacks.OnReasoningDelta(choice.Delta.ReasoningContent); err != nil {
					return nil, err
				}
			}
		}
		if choice.Delta.Content != "" {
			contentBuilder.WriteString(choice.Delta.Content)
			if callbacks.OnContentDelta != nil {
				if err := callbacks.OnContentDelta(choice.Delta.Content); err != nil {
					return nil, err
				}
			}
		}
		for _, fragment := range choice.Delta.ToolCalls {
			if err := handleFragment(fragment); err != nil {
				return nil, err
			}
		}
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			finishReason = *choice.FinishReason
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("run tool call stream: read stream: %w", err)
	}

	toolCalls := make([]domain.ToolCall, 0, len(indexOrder))
	for _, index := range indexOrder {
		toolCalls = append(toolCalls, *callsByIndex[index])
	}

	content := contentBuilder.String()
	if ContainsDSML(content) {
		parsedCalls, remainingContent := ParseDSMLToolCalls(content)
		if len(parsedCalls) > 0 {
			content = remainingContent
			if len(toolCalls) == 0 {
				toolCalls = parsedCalls
				finishReason = "tool_calls"
			}
		}
	}

	if finishReason == "" {
		if len(toolCalls) > 0 {
			finishReason = "tool_calls"
		} else {
			finishReason = "stop"
		}
	}

	return &domain.ToolCallResponse{
		FinishReason:     finishReason,
		Content:          domain.TextMessage(content),
		ReasoningContent: reasoningBuilder.String(),
		ToolCalls:        toolCalls,
		Model:            responseModel,
	}, nil
}

func sanitizeAssistantReasoning(messages []domain.ChatMessage, enableThinking bool) []domain.ChatMessage {
	sanitized := make([]domain.ChatMessage, len(messages))
	copy(sanitized, messages)
	if !enableThinking {
		return sanitized
	}
	for index := range sanitized {
		if sanitized[index].Role == "assistant" && len(sanitized[index].ToolCalls) > 0 && sanitized[index].ReasoningContent == "" {
			sanitized[index].ReasoningContent = reasoningFallbackValue
		}
	}
	return sanitized
}

func applyThinkingOptions(requestBody map[string]interface{}, enableThinking bool, reasoningEffort string) {
	if enableThinking && reasoningEffort != "" && reasoningEffort != "none" {
		requestBody[thinkingFieldName] = map[string]string{"type": thinkingModeEnabled}
		requestBody[reasoningEffortFieldName] = reasoningEffort
		requestBody[outputConfigFieldName] = map[string]string{"effort": reasoningEffort}
		return
	}
	requestBody[thinkingFieldName] = map[string]string{"type": thinkingModeDisabled}
	requestBody["temperature"] = 0.1
}

func wrapToolCallCallbacks(callbacks domain.ToolCallStreamCallbacks, outputFlushed *atomic.Bool) domain.ToolCallStreamCallbacks {
	wrapped := callbacks
	if callbacks.OnReasoningDelta != nil {
		wrapped.OnReasoningDelta = func(delta string) error {
			outputFlushed.Store(true)
			return callbacks.OnReasoningDelta(delta)
		}
	}
	if callbacks.OnContentDelta != nil {
		wrapped.OnContentDelta = func(delta string) error {
			outputFlushed.Store(true)
			return callbacks.OnContentDelta(delta)
		}
	}
	if callbacks.OnToolCallStart != nil {
		wrapped.OnToolCallStart = func(callID, name string) error {
			outputFlushed.Store(true)
			return callbacks.OnToolCallStart(callID, name)
		}
	}
	if callbacks.OnToolCallArgsDelta != nil {
		wrapped.OnToolCallArgsDelta = func(callID, delta string) error {
			outputFlushed.Store(true)
			return callbacks.OnToolCallArgsDelta(callID, delta)
		}
	}
	return wrapped
}
