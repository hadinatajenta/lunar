package domain

import (
	"context"
	"encoding/json"
)

type SourceReference struct {
	Repo      string `json:"repo"`
	File      string `json:"file"`
	Type      string `json:"type"`
	Snippet   string `json:"snippet"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

type ToolResult struct {
	Content string
	Sources []SourceReference
}

type Tool interface {
	Definition() ToolDefinition
	Execute(ctx context.Context, args json.RawMessage) (ToolResult, error)
}

type ToolRegistry interface {
	Definitions() []ToolDefinition
	ScopedDefinitions(domains []string) []ToolDefinition
	Execute(ctx context.Context, name string, args json.RawMessage) (ToolResult, error)
	Get(name string) (Tool, bool)
}
