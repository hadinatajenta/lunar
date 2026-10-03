package domain

import "time"

type ReasoningEffort string

const (
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortHigh   ReasoningEffort = "high"
)

type ChatSource struct {
	Type      string `json:"type"`
	Label     string `json:"label"`
	Repo      string `json:"repo,omitempty"`
	Path      string `json:"path,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

type TranscriptSource struct {
	Repo      string `json:"repo"`
	File      string `json:"file"`
	Path      string `json:"path"`
	Type      string `json:"type"`
	Snippet   string `json:"snippet"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type TranscriptRequest struct {
	SessionID  string             `json:"session_id"`
	Question   string             `json:"question"`
	Answer     string             `json:"answer"`
	Reasoning  string             `json:"reasoning"`
	Sources    []TranscriptSource `json:"sources"`
	DurationMs int                `json:"duration_ms"`
}

type TranscriptResponse struct {
	MessageID string `json:"message_id"`
}

type ChatMessage struct {
	ID                 string       `json:"id"`
	ChatID             string       `json:"chat_id"`
	Role               string       `json:"role"`
	Content            string       `json:"content"`
	Reasoning          string       `json:"reasoning,omitempty"`
	ThinkingDurationMs int          `json:"thinking_duration_ms,omitempty"`
	Sources            []ChatSource `json:"sources,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
}

type ChatSession struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	Title           string          `json:"title"`
	Model           string          `json:"model"`
	ThinkingMode    bool            `json:"thinking_mode"`
	ReasoningEffort ReasoningEffort `json:"reasoning_effort"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type ChatRequest struct {
	SessionID       string          `json:"session_id"`
	Model           string          `json:"model"`
	Prompt          string          `json:"prompt"`
	ThinkingMode    bool            `json:"thinking_mode"`
	ReasoningEffort ReasoningEffort `json:"reasoning_effort"`
	ActiveTools     []string        `json:"active_tools"`
}

type ChatResponse struct {
	SessionID string      `json:"session_id"`
	Message   ChatMessage `json:"message"`
}

type ModelInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Description  string `json:"description"`
	HasThinking  bool   `json:"has_thinking"`
	IsConfigured bool   `json:"is_configured"`
}
