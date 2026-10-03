package application

import "strings"

var dsmlMarkers = []string{"<｜｜DSML", "<｜DSML", "<||DSML", "<|DSML"}

type contentFilter struct {
	held       string
	suppressed bool
}

func newContentFilter() *contentFilter {
	return &contentFilter{}
}

func (f *contentFilter) accept(delta string) string {
	if f.suppressed {
		return ""
	}

	combined := f.held + delta
	f.held = ""

	if index := indexOfMarker(combined); index >= 0 {
		f.suppressed = true
		return combined[:index]
	}

	heldLength := trailingPartialMarkerLength(combined)
	if heldLength > 0 {
		f.held = combined[len(combined)-heldLength:]
		return combined[:len(combined)-heldLength]
	}
	return combined
}

func indexOfMarker(text string) int {
	found := -1
	for _, marker := range dsmlMarkers {
		index := strings.Index(text, marker)
		if index < 0 {
			continue
		}
		if found < 0 || index < found {
			found = index
		}
	}
	return found
}

func trailingPartialMarkerLength(text string) int {
	longest := 0
	for _, marker := range dsmlMarkers {
		limit := len(marker) - 1
		if len(text) < limit {
			limit = len(text)
		}
		for length := limit; length > longest; length-- {
			if strings.HasSuffix(text, marker[:length]) {
				longest = length
				break
			}
		}
	}
	return longest
}
