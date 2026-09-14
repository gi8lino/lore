package main

import (
	"strings"
	"testing"

	"github.com/gi8lino/lore/pluginapi"
)

// TestTransformHighlightsKnownLanguage verifies Chroma owns bundled highlighting.
func TestTransformHighlightsKnownLanguage(t *testing.T) {
	t.Parallel()

	result := transform(pluginapi.RenderRequest{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "go", Source: "package main\n"})
	if result.Error != "" {
		t.Fatalf("transform returned error: %s", result.Error)
	}
	if !result.Matched || len(result.Parts) != 1 || !strings.Contains(result.Parts[0].Text, `class="chroma"`) {
		t.Fatalf("unexpected highlight result: %#v", result)
	}
}

// TestTransformLeavesUnknownLanguageUnmatched verifies another renderer can fall back safely.
func TestTransformLeavesUnknownLanguageUnmatched(t *testing.T) {
	t.Parallel()

	result := transform(pluginapi.RenderRequest{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "not-a-real-language", Source: "text"})
	if result.Error != "" || result.Matched || len(result.Parts) != 0 {
		t.Fatalf("unexpected unmatched result: %#v", result)
	}
}
