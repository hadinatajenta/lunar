package application

import (
	"strings"
	"testing"
)

func assertEverySuggestionHasAReason(t *testing.T, suggestions *RepositorySuggestions) {
	t.Helper()

	for _, suggestion := range suggestions.Suggestions {
		if len(suggestion.Reasons) == 0 {
			t.Fatalf("a suggestion without a reason is a defect: %s", suggestion.Repo)
		}
	}
}

func assertReasonInvariants(t *testing.T, suggestions *RepositorySuggestions) {
	t.Helper()

	for _, suggestion := range suggestions.Suggestions {
		for _, reason := range suggestion.Reasons {
			weight, known := signalWeights[reason.Signal]
			if !known {
				t.Fatalf("unknown signal %q in %s", reason.Signal, suggestion.Repo)
			}
			if reason.Weight != weight {
				t.Fatalf("signal %q carries weight %v, expected %v", reason.Signal, reason.Weight, weight)
			}
			if reason.Strength <= 0 || reason.Strength > 1 {
				t.Fatalf("signal %q carries an out of range strength %v", reason.Signal, reason.Strength)
			}
			if strings.TrimSpace(reason.Text) == "" {
				t.Fatalf("signal %q carries no text", reason.Signal)
			}
		}
	}
}

func findReason(suggestion RepositorySuggestion, signal string) (SuggestionReason, bool) {
	for _, reason := range suggestion.Reasons {
		if reason.Signal == signal {
			return reason, true
		}
	}

	return SuggestionReason{}, false
}

func findUnavailable(suggestions *RepositorySuggestions, signal string) (UnavailableSignal, bool) {
	for _, unavailable := range suggestions.SignalsUnavailable {
		if unavailable.Signal == signal {
			return unavailable, true
		}
	}

	return UnavailableSignal{}, false
}
