package infrastructure

import "testing"

func TestClaudeOutputTokenCapStaysInsideTheModelLimit(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		want      int
	}{
		{name: "unset falls back to the default", requested: 0, want: claudeDefaultMaxTokens},
		{name: "negative falls back to the default", requested: -100, want: claudeDefaultMaxTokens},
		{name: "a request below the limit is honoured", requested: 6000, want: 6000},
		{name: "a reasoning budget is clamped", requested: reviewThinkingOutputTokens, want: claudeMaxOutputTokens},
		{name: "a request far above the limit is clamped", requested: 200000, want: claudeMaxOutputTokens},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := claudeOutputTokenCap(testCase.requested); got != testCase.want {
				t.Errorf("claudeOutputTokenCap(%d) = %d, want %d", testCase.requested, got, testCase.want)
			}
		})
	}
}

func TestReviewOutputTokenCapStaysBelowTheDeepSeekReasoningDefault(t *testing.T) {
	deepSeekReasoningDefault := 64000
	if reviewOutputTokenCap(true) >= deepSeekReasoningDefault {
		t.Errorf("the reasoning cap %d must stay inside the provider default of %d", reviewOutputTokenCap(true), deepSeekReasoningDefault)
	}
	if reviewOutputTokenCap(true) <= reviewNonThinkingOutputTokens {
		t.Error("a reasoning review must be allowed more output than a chat review")
	}
	if reviewOutputTokenCap(false) != reviewNonThinkingOutputTokens {
		t.Errorf("the chat cap = %d, want %d", reviewOutputTokenCap(false), reviewNonThinkingOutputTokens)
	}
}
