package domain

import (
	"strings"
	"testing"
)

func TestResolveOpenCodeGoModelMatchesEveryIdentifier(t *testing.T) {
	if len(OpenCodeGoCatalog) != 3 {
		t.Fatalf("catalog size = %d, want 3", len(OpenCodeGoCatalog))
	}

	expected := []struct {
		catalogID  string
		name       string
		upstreamID string
	}{
		{catalogID: "opencode-go-deepseek-v4.1-flash", name: "DeepSeek V4.1 Flash (Go)", upstreamID: "deepseek-v4.1-flash"},
		{catalogID: "opencode-go-mimo-v2.6-flash", name: "MiMo V2.6 Flash (Go)", upstreamID: "mimo-v2.6-flash"},
		{catalogID: "opencode-go-space-bunny-free", name: "Space Bunny Free (Go)", upstreamID: "space-bunny-free"},
	}

	for index, want := range expected {
		entry := OpenCodeGoCatalog[index]
		if entry.CatalogID != want.catalogID || entry.Name != want.name || entry.UpstreamID != want.upstreamID {
			t.Fatalf("catalog[%d] = %+v, want catalog id %q name %q upstream id %q", index, entry, want.catalogID, want.name, want.upstreamID)
		}
		if strings.TrimSpace(entry.Description) == "" {
			t.Errorf("catalog[%d] description is empty", index)
		}
	}

	for _, entry := range OpenCodeGoCatalog {
		identifiers := []string{entry.CatalogID, entry.Name, entry.UpstreamID, "  " + strings.ToUpper(entry.Name) + "  "}
		for _, identifier := range identifiers {
			resolved, found := ResolveOpenCodeGoModel(identifier)
			if !found {
				t.Errorf("ResolveOpenCodeGoModel(%q) not found, want %q", identifier, entry.UpstreamID)
				continue
			}
			if resolved != entry {
				t.Errorf("ResolveOpenCodeGoModel(%q) = %+v, want %+v", identifier, resolved, entry)
			}
		}
	}
}

func TestResolveOpenCodeGoModelRejectsUnknownIdentifiers(t *testing.T) {
	identifiers := []string{"", "   ", "deepseek-v4-pro", "DeepSeek-V4 Pro (Thinking)", "mimo-v2.5-pro", "space-bunny", "opencode-go"}

	for _, identifier := range identifiers {
		resolved, found := ResolveOpenCodeGoModel(identifier)
		if found {
			t.Errorf("ResolveOpenCodeGoModel(%q) = %+v, want not found", identifier, resolved)
		}
	}
}
