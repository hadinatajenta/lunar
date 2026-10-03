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

	"lunar/backend/internal/agent/domain"
)

type ReasoningStreamer interface {
	StreamAnswerWithReasoning(
		ctx context.Context,
		messages []domain.ChatMessage,
		model string,
		onReasoning func(chunk string) error,
		onChunk func(chunk string) error,
	) error
}

func (c *ChatClient) resolveModel(requested string) (string, bool, string) {
	option := strings.TrimSpace(requested)
	if option == "" {
		option = strings.TrimSpace(c.cfg.Model)
	}
	if option == "" {
		return defaultChatModel, false, "none"
	}

	switch strings.ToLower(option) {
	case "deepseek-flash-low", "flash-low":
		return "deepseek-flash", true, "low"
	case "deepseek-flash-thinking", "flash-thinking", "thinking", "deepseek-flash-high", "flash-high":
		return "deepseek-flash", true, "high"
	case "deepseek-flash-max", "flash-max":
		return "deepseek-flash", true, "max"
	case "deepseek-v4-pro", "v4-pro", "pro", "deepseek-pro":
		return "deepseek-v4-pro", true, "high"
	case "deepseek-v4-pro-direct":
		return "deepseek-v4-pro", false, "none"
	case "deepseek-flash", "flash", "deepseek-chat":
		return "deepseek-flash", false, "none"
	default:
		return option, false, "none"
	}
}

func (c *ChatClient) buildToolCallRequestBody(
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	modelOption string,
	toolChoice string,
	stream bool,
) map[string]interface{} {
	actualModel, enableThinking, reasoningEffort := c.resolveModel(modelOption)

	normalizedChoice := strings.ToLower(strings.TrimSpace(toolChoice))
	isStrictOrNamed := normalizedChoice == "required" ||
		(normalizedChoice != "" && normalizedChoice != "auto" && normalizedChoice != "none")
	if isStrictOrNamed {
		enableThinking = false
	}

	sanitizedMessages := sanitizeAssistantReasoning(messages, enableThinking)

	requestBody := map[string]interface{}{
		"model":    actualModel,
		"messages": sanitizedMessages,
	}

	if stream {
		requestBody["stream"] = true
	}

	if c.supportsReasoningEffort() {
		applyThinkingOptions(requestBody, enableThinking, reasoningEffort)
	} else {
		requestBody["temperature"] = 0.1
	}

	if len(tools) == 0 {
		return requestBody
	}

	requestBody["tools"] = tools
	switch normalizedChoice {
	case "", "auto":
		requestBody[toolChoiceFieldName] = "auto"
	case "required":
		requestBody[toolChoiceFieldName] = "required"
	case "none":
		requestBody[toolChoiceFieldName] = "none"
	default:
		requestBody[toolChoiceFieldName] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name": strings.TrimSpace(toolChoice),
			},
		}
	}
	return requestBody
}

func (c *ChatClient) RunToolCall(
	ctx context.Context,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	modelOption string,
	toolChoice string,
) (*domain.ToolCallResponse, error) {
	if strings.TrimSpace(c.cfg.APIKey) == "" {
		return nil, errors.New(missingAPIKeyMessage)
	}

	encoded, err := marshalRequestBody(c.buildToolCallRequestBody(messages, tools, modelOption, toolChoice, false))
	if err != nil {
		return nil, err
	}

	resp, err := c.sendJSON(ctx, c.cfg.BaseURL+chatCompletionsPath, encoded)
	if err != nil {
		return nil, fmt.Errorf("run tool call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("run tool call: read error body: %w", readErr)
		}
		return nil, parseProviderError(c.providerName(), resp.StatusCode, body)
	}

	var chatResponse struct {
		Model   string `json:"model"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content          domain.MessageContent `json:"content"`
				ReasoningContent string                `json:"reasoning_content"`
				ToolCalls        []domain.ToolCall     `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		return nil, fmt.Errorf("run tool call: decode response: %w", err)
	}
	if len(chatResponse.Choices) == 0 {
		return nil, errors.New(emptyChoicesMessage)
	}

	choice := chatResponse.Choices[0]
	toolCalls := choice.Message.ToolCalls
	content := choice.Message.Content
	finishReason := choice.FinishReason

	if ContainsDSML(content.Text()) {
		parsedCalls, remainingContent := ParseDSMLToolCalls(content.Text())
		if len(parsedCalls) > 0 {
			content = domain.TextMessage(remainingContent)
			if len(toolCalls) == 0 {
				toolCalls = parsedCalls
				finishReason = "tool_calls"
			}
		}
	}

	return &domain.ToolCallResponse{
		FinishReason:     finishReason,
		Content:          content,
		ReasoningContent: choice.Message.ReasoningContent,
		ToolCalls:        toolCalls,
		Model:            chatResponse.Model,
	}, nil
}

func (c *ChatClient) StreamAnswer(
	ctx context.Context,
	messages []domain.ChatMessage,
	modelOption string,
	onChunk func(chunk string) error,
) error {
	return c.StreamAnswerWithReasoning(ctx, messages, modelOption, nil, onChunk)
}

func (c *ChatClient) StreamAnswerWithReasoning(
	ctx context.Context,
	messages []domain.ChatMessage,
	modelOption string,
	onReasoning func(chunk string) error,
	onChunk func(chunk string) error,
) error {
	if strings.TrimSpace(c.cfg.APIKey) == "" {
		return errors.New(missingAPIKeyMessage)
	}

	actualModel, enableThinking, reasoningEffort := c.resolveModel(modelOption)
	sanitizedMessages := sanitizeAssistantReasoning(messages, enableThinking)

	requestBody := map[string]interface{}{
		"model":    actualModel,
		"stream":   true,
		"messages": sanitizedMessages,
	}
	if c.supportsReasoningEffort() {
		applyThinkingOptions(requestBody, enableThinking, reasoningEffort)
	} else {
		requestBody["temperature"] = 0.1
	}

	encoded, err := marshalRequestBody(requestBody)
	if err != nil {
		return err
	}

	resp, err := c.sendJSON(ctx, c.cfg.BaseURL+chatCompletionsPath, encoded)
	if err != nil {
		return fmt.Errorf("stream answer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("stream answer: read error body: %w", readErr)
		}
		return parseProviderError(c.providerName(), resp.StatusCode, body)
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
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		if chunk.Choices[0].Delta.ReasoningContent != "" && onReasoning != nil {
			if err := onReasoning(chunk.Choices[0].Delta.ReasoningContent); err != nil {
				return err
			}
		}
		if chunk.Choices[0].Delta.Content != "" && onChunk != nil {
			if err := onChunk(chunk.Choices[0].Delta.Content); err != nil {
				return err
			}
		}
		if chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason == "stop" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream answer: read stream: %w", err)
	}
	return nil
}
