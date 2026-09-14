package main

import (
	"strings"
	"testing"

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

func TestCuratedLexerRegistry(t *testing.T) {
	for _, language := range curatedLexerNames {
		if lexer := highlightLexers.Get(language); lexer == nil {
			t.Errorf("expected %q lexer to be available", language)
		}
	}

	for _, alias := range []string{"sh", "cpp", "csharp", "dockerfile", "js", "make", "md", "nginx", "ps1", "py", "rb", "rs", "tf", "ts", "yml"} {
		if lexer := highlightLexers.Get(alias); lexer == nil {
			t.Errorf("expected %q alias to be available", alias)
		}
	}

	if lexer := highlightLexers.Get("brainfuck"); lexer != nil {
		t.Fatalf("unexpected unselected lexer: %s", lexer.Config().Name)
	}
}

func TestCuratedRegistryRetainsLexerInstances(t *testing.T) {
	bash := highlightLexers.Get("bash")
	if bash == nil {
		t.Fatal("bash lexer is unavailable")
	}
	if bash != highlightLexers.Get("sh") || bash != highlightLexers.Get("shell") || bash != highlightLexers.Get("bash") {
		t.Fatal("bash aliases did not reuse the curated lexer instance")
	}
}

func TestTransformLeavesUnknownLanguageUnmatched(t *testing.T) {
	result := transform(pluginapi.RenderRequest{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: "not-a-real-language", Source: "text"})
	if result.Error != "" || result.Matched || len(result.Parts) != 0 {
		t.Fatalf("unexpected unmatched result: %#v", result)
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
