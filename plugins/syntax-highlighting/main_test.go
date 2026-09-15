package main

import (
	"strings"
	"testing"

	"github.com/gi8lino/lore/pluginapi"
)

func TestTransformHighlightsKnownLanguage(t *testing.T) {
	result := transform(request("go", "package main\n\nfunc main() { println(\"Lore\") }\n"))
	if result.Error != "" {
		t.Fatalf("transform returned error: %s", result.Error)
	}
	if !result.Matched || len(result.Parts) != 1 {
		t.Fatalf("unexpected highlight result: %#v", result)
	}
	for _, expected := range []string{`class="chroma"`, `class="kd"`, `class="nf"`, `&#34;Lore&#34;`} {
		if !strings.Contains(result.Parts[0].Text, expected) {
			t.Fatalf("highlighted output missing %q: %s", expected, result.Parts[0].Text)
		}
	}
}

func TestTransformHighlightsFenceAliases(t *testing.T) {
	for _, alias := range []string{"sh", "cpp", "csharp", "dockerfile", "js", "make", "md", "nginx", "ps1", "py", "rb", "rs", "tf", "ts", "yml"} {
		result := transform(request(alias, sampleFor(alias)))
		if result.Error != "" || !result.Matched {
			t.Errorf("alias %q was not highlighted: %#v", alias, result)
		}
	}
}

func TestLanguageRegistryRetainsCurrentCoverage(t *testing.T) {
	for _, name := range []string{
		"bash", "c", "cpp", "csharp", "css", "dockerfile", "go", "hcl", "html", "java", "javascript", "json", "kotlin", "lua", "makefile", "markdown", "nginx", "php", "powershell", "python", "ruby", "rust", "sql", "terraform", "toml", "typescript", "xml", "yaml", "zig",
	} {
		if _, ok := lookupLanguage(name); !ok {
			t.Errorf("expected %q language to be available", name)
		}
	}
	if _, ok := lookupLanguage("brainfuck"); ok {
		t.Fatal("unexpected unsupported language")
	}
}

func TestEveryLanguageScannerRenders(t *testing.T) {
	for _, item := range languageDefinitions() {
		if len(item.aliases) == 0 {
			t.Fatalf("language %q has no aliases", item.name)
		}
		result := transform(request(item.aliases[0], "name = \"value\" // comment\n"))
		if result.Error != "" || !result.Matched || len(result.Parts) != 1 {
			t.Errorf("language %q failed to render: %#v", item.name, result)
		}
	}
}

func TestGoTokenClasses(t *testing.T) {
	html := transform(request("go", "// comment\nvar total = 42\nfunc sum(value int) string { return \"ok\" }\n")).Parts[0].Text
	for _, expected := range []string{`class="c1"`, `class="kd"`, `class="mi"`, `class="nf"`, `class="kt"`, `class="k"`, `class="s2"`} {
		if !strings.Contains(html, expected) {
			t.Errorf("output missing %s: %s", expected, html)
		}
	}
}

func TestMarkupScannerEscapesSource(t *testing.T) {
	result := transform(request("html", `<script data-x="1">alert("x")</script>`))
	if result.Error != "" || !result.Matched {
		t.Fatalf("unexpected result: %#v", result)
	}
	output := result.Parts[0].Text
	if strings.Contains(output, "<script data-x") {
		t.Fatalf("source was emitted as executable markup: %s", output)
	}
	for _, expected := range []string{"&lt;", `class="nt">script`, `class="na">data-x`, "&gt;"} {
		if !strings.Contains(output, expected) {
			t.Errorf("markup output missing %q: %s", expected, output)
		}
	}
}

func TestTransformLeavesUnknownLanguageUnmatched(t *testing.T) {
	result := transform(request("not-a-real-language", "text"))
	if result.Error != "" || result.Matched || len(result.Parts) != 0 {
		t.Fatalf("unexpected unmatched result: %#v", result)
	}
}

func TestTransformRejectsWrongModuleOrStage(t *testing.T) {
	for _, input := range []pluginapi.RenderRequest{
		{APIVersion: pluginapi.Version, Module: "other", Stage: "highlight", Language: "go"},
		{APIVersion: pluginapi.Version, Module: "chroma", Stage: "other", Language: "go"},
	} {
		result := transform(input)
		if result.Error == "" {
			t.Fatalf("expected request rejection: %#v", input)
		}
	}
}

func BenchmarkTransformWarmBash(b *testing.B) {
	input := request("bash", "set -euo pipefail\necho \"Lore\"\n")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := transform(input)
		if result.Error != "" || !result.Matched {
			b.Fatalf("unexpected result: %#v", result)
		}
	}
}

func request(language, source string) pluginapi.RenderRequest {
	return pluginapi.RenderRequest{APIVersion: pluginapi.Version, Module: "chroma", Stage: "highlight", Language: language, Source: source}
}

func sampleFor(language string) string {
	switch language {
	case "sh":
		return "echo $HOME\n"
	case "md":
		return "# Heading\n"
	case "dockerfile":
		return "FROM scratch\n"
	case "yml":
		return "enabled: true\n"
	default:
		return "value = 1\n"
	}
}
