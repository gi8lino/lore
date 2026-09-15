package main

import (
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/gi8lino/lore/pluginapi"
)

func TestTransformHighlightsKnownLanguage(t *testing.T) {
	result := transform(pluginapi.RenderRequest{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "go", Source: "package main\n"})
	if result.Error != "" {
		t.Fatalf("transform returned error: %s", result.Error)
	}
	if !result.Matched || len(result.Parts) != 1 || !strings.Contains(result.Parts[0].Text, `class="chroma"`) {
		t.Fatalf("unexpected highlight result: %#v", result)
	}
}

func TestTransformHighlightsFenceAlias(t *testing.T) {
	result := transform(pluginapi.RenderRequest{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "sh", Source: "echo Lore\n"})
	if result.Error != "" || !result.Matched {
		t.Fatalf("unexpected alias result: %#v", result)
	}
}

func TestFullChromaRegistryIsAvailable(t *testing.T) {
	for _, language := range []string{"go", "typescript", "yaml", "brainfuck"} {
		if lexer := lexers.Get(language); lexer == nil {
			t.Errorf("expected %q lexer to be available", language)
		}
	}
	if len(lexers.Names(false)) < 200 {
		t.Fatalf("expected full Chroma lexer registry, got %d lexers", len(lexers.Names(false)))
	}
}

func TestTransformNeverAutoDetectsLanguage(t *testing.T) {
	for _, language := range []string{"", "   ", "not-a-real-language"} {
		result := transform(pluginapi.RenderRequest{
			APIVersion: pluginapi.Version,
			Module:     "chroma",
			Stage:      "highlight",
			Language:   language,
			Source:     "package main\n\nfunc main() {}\n",
		})
		if result.Error != "" || result.Matched || len(result.Parts) != 0 {
			t.Fatalf("language %q must stay unmatched instead of being auto-detected: %#v", language, result)
		}
	}
}

func TestTransformUsesLanguagePerCodeBlock(t *testing.T) {
	requests := []pluginapi.RenderRequest{
		{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "go", Source: "package main\nfunc main() {}\n"},
		{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "python", Source: "def main():\n    pass\n"},
		{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "sql", Source: "SELECT * FROM users;\n"},
	}

	for _, request := range requests {
		result := transform(request)
		if result.Error != "" || !result.Matched || len(result.Parts) != 1 {
			t.Fatalf("language %q did not use its explicit lexer: %#v", request.Language, result)
		}
	}
}

func TestTransformLeavesUnknownLanguageUnmatched(t *testing.T) {
	result := transform(pluginapi.RenderRequest{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "not-a-real-language", Source: "text"})
	if result.Error != "" || result.Matched || len(result.Parts) != 0 {
		t.Fatalf("unexpected unmatched result: %#v", result)
	}
}

func TestTransformRejectsUnsupportedRequest(t *testing.T) {
	for _, request := range []pluginapi.RenderRequest{
		{APIVersion: pluginapi.Version, Module: "other", Stage: "highlight", Language: "go", Source: "package main\n"},
		{APIVersion: pluginapi.Version, Module: "chroma", Stage: "other", Language: "go", Source: "package main\n"},
	} {
		result := transform(request)
		if result.Error == "" || result.Matched {
			t.Fatalf("unexpected unsupported request result: %#v", result)
		}
	}
}

func BenchmarkTransformWarmBash(b *testing.B) {
	request := pluginapi.RenderRequest{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "bash", Source: "set -euo pipefail\necho \"Lore\"\n"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := transform(request)
		if result.Error != "" || !result.Matched {
			b.Fatalf("unexpected result: %#v", result)
		}
	}
}
