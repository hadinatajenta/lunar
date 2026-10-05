package domain

import "strings"

const OpenCodeGoProvider = "opencode-go"

type OpenCodeGoModel struct {
	CatalogID   string
	Name        string
	UpstreamID  string
	Description string
}

var OpenCodeGoCatalog = []OpenCodeGoModel{
	{
		CatalogID:   "opencode-go-deepseek-v4.1-flash",
		Name:        "DeepSeek V4.1 Flash (Go)",
		UpstreamID:  "deepseek-v4.1-flash",
		Description: "Fast DeepSeek chat model served through the OpenCode Go subscription.",
	},
	{
		CatalogID:   "opencode-go-mimo-v2.6-flash",
		Name:        "MiMo V2.6 Flash (Go)",
		UpstreamID:  "mimo-v2.6-flash",
		Description: "Low-latency MiMo chat model served through the OpenCode Go subscription.",
	},
	{
		CatalogID:   "opencode-go-space-bunny-free",
		Name:        "Space Bunny Free (Go)",
		UpstreamID:  "space-bunny-free",
		Description: "Free-tier OpenCode Go model for lightweight general requests.",
	},
}

func ResolveOpenCodeGoModel(identifier string) (OpenCodeGoModel, bool) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return OpenCodeGoModel{}, false
	}

	for _, model := range OpenCodeGoCatalog {
		if strings.EqualFold(trimmed, model.CatalogID) ||
			strings.EqualFold(trimmed, model.Name) ||
			strings.EqualFold(trimmed, model.UpstreamID) {
			return model, true
		}
	}

	return OpenCodeGoModel{}, false
}
