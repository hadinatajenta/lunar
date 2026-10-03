package domain

import (
	"context"
	"encoding/json"
	"strings"
)

type ContentPartType string

const (
	ContentPartTypeText ContentPartType = "text"
	ContentPartTypeFile ContentPartType = "file"
)

type ContentPart interface {
	isContentPart()
}

type TextPart struct {
	Type ContentPartType `json:"type"`
	Text string          `json:"text"`
}

func (TextPart) isContentPart() {}

type FilePart struct {
	Type   ContentPartType `json:"type"`
	FileID string          `json:"file_id"`
}

func (FilePart) isContentPart() {}

type MessageContent struct {
	String *string
	Parts  []ContentPart
}

func (mc MessageContent) MarshalJSON() ([]byte, error) {
	if mc.String != nil {
		return json.Marshal(*mc.String)
	}
	if mc.Parts != nil {
		return json.Marshal(mc.Parts)
	}
	return json.Marshal("")
}

func (mc *MessageContent) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		mc.String = &text
		return nil
	}

	var parts []json.RawMessage
	if err := json.Unmarshal(data, &parts); err != nil {
		return nil
	}

	mc.Parts = make([]ContentPart, len(parts))
	for index, raw := range parts {
		var partType struct {
			Type ContentPartType `json:"type"`
		}
		if err := json.Unmarshal(raw, &partType); err != nil {
			return err
		}
		switch partType.Type {
		case ContentPartTypeText:
			var textPart TextPart
			if err := json.Unmarshal(raw, &textPart); err != nil {
				return err
			}
			mc.Parts[index] = textPart
		case ContentPartTypeFile:
			var filePart FilePart
			if err := json.Unmarshal(raw, &filePart); err != nil {
				return err
			}
			mc.Parts[index] = filePart
		default:
			return nil
		}
	}
	return nil
}

func (mc MessageContent) Text() string {
	if mc.String != nil {
		return *mc.String
	}
	if mc.Parts == nil {
		return ""
	}
	var builder strings.Builder
	for _, part := range mc.Parts {
		if textPart, ok := part.(TextPart); ok {
			builder.WriteString(textPart.Text)
		}
	}
	return builder.String()
}

func (mc MessageContent) IsEmpty() bool {
	return mc.Text() == ""
}

func TextMessage(text string) MessageContent {
	return MessageContent{String: &text}
}

type ChatMessage struct {
	Role             string         `json:"role"`
	Content          MessageContent `json:"content"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
}

type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      bool            `json:"strict,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCallResponse struct {
	FinishReason     string         `json:"finish_reason"`
	Content          MessageContent `json:"content,omitempty"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	Model            string         `json:"model"`
}

type ToolCallStreamCallbacks struct {
	OnReasoningDelta    func(delta string) error
	OnContentDelta      func(delta string) error
	OnToolCallStart     func(callID, name string) error
	OnToolCallArgsDelta func(callID, delta string) error
}

type AIClient interface {
	RunToolCall(ctx context.Context, messages []ChatMessage, tools []ToolDefinition, model string, toolChoice string) (*ToolCallResponse, error)
	StreamAnswer(ctx context.Context, messages []ChatMessage, model string, onChunk func(chunk string) error) error
}

type StreamToolCaller interface {
	RunToolCallStream(
		ctx context.Context,
		messages []ChatMessage,
		tools []ToolDefinition,
		model string,
		toolChoice string,
		callbacks ToolCallStreamCallbacks,
	) (*ToolCallResponse, error)
}

type FailoverHook func(ctx context.Context, from string, to string, reason string, targetModel string)

type failoverContextKey struct{}

func WithFailoverHook(ctx context.Context, hook FailoverHook) context.Context {
	return context.WithValue(ctx, failoverContextKey{}, hook)
}

func FailoverHookFromContext(ctx context.Context) FailoverHook {
	if hook, ok := ctx.Value(failoverContextKey{}).(FailoverHook); ok {
		return hook
	}
	return nil
}
