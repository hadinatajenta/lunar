package infrastructure

import "testing"

func TestChatClient_SupportsImageInputOnlyForCapableModels(t *testing.T) {
	client := NewChatClient(ProviderConfig{})

	cases := map[string]bool{
		"deepseek-flash":             true,
		"deepseek-chat":              true,
		"deepseek-flash-thinking":    true,
		"deepseek-reasoner":          false,
		"deepseek-v4-pro":            false,
		"DeepSeek-V4 Pro (Thinking)": false,
		"gpt-4o":                     true,
		"gpt-6-astra":                false,
		"claude-3-5-sonnet-20241022": true,
		"claude-3-7-sonnet-20250219": true,
		"gemini-1.5-pro":             true,
		"mimo-v2.5-pro":              false,
	}

	for modelOption, expected := range cases {
		if actual := client.SupportsImageInput(modelOption); actual != expected {
			t.Errorf("expected SupportsImageInput(%q) to be %t, got %t", modelOption, expected, actual)
		}
	}
}
