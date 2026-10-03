package application

import "testing"

func TestContentFilterPassesPlainText(t *testing.T) {
	filter := newContentFilter()

	first := filter.accept("The late fee is calculated in ")
	second := filter.accept("the settlement calculator.")

	if first+second != "The late fee is calculated in the settlement calculator." {
		t.Fatalf("plain text was altered: %q", first+second)
	}
}

func TestContentFilterSuppressesDSMLBlock(t *testing.T) {
	filter := newContentFilter()

	visible := filter.accept("Before text <｜｜DSML｜｜tool_calls><｜｜DSML｜｜invoke name=\"search_code\">")
	trailing := filter.accept("more markup that must never reach the client")

	if visible != "Before text " {
		t.Fatalf("expected the text before the marker, got %q", visible)
	}
	if trailing != "" {
		t.Fatalf("expected everything after the marker to be suppressed, got %q", trailing)
	}
}

func TestContentFilterHandlesMarkerSplitAcrossDeltas(t *testing.T) {
	filter := newContentFilter()

	var visible string
	for _, delta := range []string{"Answer ", "<｜｜DS", "ML｜｜tool_calls>", "hidden"} {
		visible += filter.accept(delta)
	}

	if visible != "Answer " {
		t.Fatalf("expected only the answer prefix, got %q", visible)
	}
}

func TestContentFilterHandlesASCIIMarker(t *testing.T) {
	filter := newContentFilter()

	visible := filter.accept("Real answer <||DSML||tool_calls>")

	if visible != "Real answer " {
		t.Fatalf("expected the ASCII marker to be suppressed, got %q", visible)
	}
}

func TestContentFilterReleasesHeldTextThatIsNotAMarker(t *testing.T) {
	filter := newContentFilter()

	held := filter.accept("The result is <")
	released := filter.accept("strong>")

	if held != "The result is " {
		t.Fatalf("expected the partial marker to be held, got %q", held)
	}
	if released != "<strong>" {
		t.Fatalf("expected the held text to be released once it proved harmless, got %q", released)
	}
}

func TestContentFilterStaysSilentAfterSuppression(t *testing.T) {
	filter := newContentFilter()

	filter.accept("text <｜｜DSML")
	if remaining := filter.accept("anything at all"); remaining != "" {
		t.Fatalf("expected the filter to remain suppressed, got %q", remaining)
	}
}
