package infrastructure

import (
	"strings"
	"testing"

	"lunar/backend/internal/copilot/domain"
)

func TestLLMClientResolveProviderAndModelRoutesOpenCodeGo(t *testing.T) {
	client := NewLLMClient("", "")

	tests := []struct {
		modelName    string
		wantProvider string
		wantModel    string
	}{
		{modelName: "DeepSeek V4.1 Flash (Go)", wantProvider: domain.OpenCodeGoProvider, wantModel: "deepseek-v4.1-flash"},
		{modelName: "opencode-go-deepseek-v4.1-flash", wantProvider: domain.OpenCodeGoProvider, wantModel: "deepseek-v4.1-flash"},
		{modelName: "deepseek-v4.1-flash", wantProvider: domain.OpenCodeGoProvider, wantModel: "deepseek-v4.1-flash"},
		{modelName: "  MiMo V2.6 Flash (Go)  ", wantProvider: domain.OpenCodeGoProvider, wantModel: "mimo-v2.6-flash"},
		{modelName: "opencode-go-mimo-v2.6-flash", wantProvider: domain.OpenCodeGoProvider, wantModel: "mimo-v2.6-flash"},
		{modelName: "space-bunny-free", wantProvider: domain.OpenCodeGoProvider, wantModel: "space-bunny-free"},
		{modelName: "opencode-go-space-bunny-free", wantProvider: domain.OpenCodeGoProvider, wantModel: "space-bunny-free"},
		{modelName: "DeepSeek-V4 Pro (Thinking)", wantProvider: "deepseek", wantModel: "deepseek-reasoner"},
		{modelName: "DeepSeek Flash", wantProvider: "deepseek", wantModel: "deepseek-chat"},
		{modelName: "Xiaomi MiMo-V2.5 Pro", wantProvider: "mimo", wantModel: "mimo-v2.5-pro"},
	}

	for _, testCase := range tests {
		t.Run(testCase.modelName, func(t *testing.T) {
			provider, actualModel, hasThinking := client.ResolveProviderAndModel(testCase.modelName)
			if provider != testCase.wantProvider {
				t.Errorf("provider = %q, want %q", provider, testCase.wantProvider)
			}
			if actualModel != testCase.wantModel {
				t.Errorf("model = %q, want %q", actualModel, testCase.wantModel)
			}
			if provider == domain.OpenCodeGoProvider && hasThinking {
				t.Error("opencode-go models must not report thinking support")
			}
		})
	}
}

func TestLLMClientProviderBaseURLForOpenCodeGo(t *testing.T) {
	client := NewLLMClient("", "")

	for _, provider := range []string{"opencode-go", "OpenCode-Go", "  opencode-go  "} {
		baseURL, err := client.ProviderBaseURL(provider)
		if err != nil {
			t.Fatalf("ProviderBaseURL(%q) error: %v", provider, err)
		}
		if baseURL != "https://opencode.ai/zen/go/v1" {
			t.Errorf("base URL = %q, want %q", baseURL, "https://opencode.ai/zen/go/v1")
		}
		if strings.HasPrefix(baseURL, "https://opencode.ai/zen/v1") {
			t.Errorf("base URL %q must not resolve to the separate zen/v1 product", baseURL)
		}
		if strings.Contains(baseURL, "/zen/v1") {
			t.Errorf("base URL %q contains the zen/v1 path", baseURL)
		}
	}
}
