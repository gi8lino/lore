// Package pluginapi contains the versioned wire types shared by Lore and WASM
// plugins. It intentionally has no dependencies on Lore's internal packages.
package pluginapi

const Version = 1

// RenderRequest invokes one declared renderer module. Features contains only
// presentation preferences, never credentials or implicit host capabilities.
type RenderRequest struct {
	APIVersion int             `json:"api_version"`
	Module     string          `json:"module"`
	Stage      string          `json:"stage"`
	Source     string          `json:"source"`
	Features   map[string]bool `json:"features,omitempty"`
}

// RenderResult returns intermediate output or an error. Markdown fragments are
// rendered by the host after the WASM call finishes; this avoids reentrant guest
// calls for nested blocks. Postprocessors may return text fragments only.
type RenderResult struct {
	Parts []RenderPart `json:"parts,omitempty"`
	Error string       `json:"error,omitempty"`
}

// RenderPart is either literal intermediate output or a recursive Markdown
// fragment. Text and Markdown must not both be set. Empty text is valid.
type RenderPart struct {
	Text     string  `json:"text,omitempty"`
	Markdown *string `json:"markdown,omitempty"`
}
