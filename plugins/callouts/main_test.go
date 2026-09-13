package main

import (
	"strings"
	"testing"

	"github.com/gi8lino/lore/pluginapi"
)

func TestCalloutFragments(t *testing.T) {
	result := transform(pluginapi.RenderRequest{APIVersion: 1, Module: "callouts", Stage: "preprocess", Source: "Before\n\n!!! warning\n**Body**\n\nAfter\n"})
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	if len(result.Parts) != 3 || result.Parts[1].Markdown == nil || *result.Parts[1].Markdown != "**Body**" {
		t.Fatalf("unexpected fragments: %+v", result.Parts)
	}
	if !strings.Contains(result.Parts[0].Text, `<aside class="callout warning">`) || !strings.Contains(result.Parts[2].Text, "After\n") {
		t.Fatalf("unexpected markup: %+v", result.Parts)
	}
}
func TestCodeAndOrdinaryMarkdownRemainLiteral(t *testing.T) {
	for _, source := range []string{"plain\n", "````\n```\n!!! warning\nBody\n`````\n", "~~~\n!!! note\nBody\n~~~", "!!! unsupported\nBody\n"} {
		result := transform(pluginapi.RenderRequest{APIVersion: 1, Module: "callouts", Stage: "preprocess", Source: source})
		if len(result.Parts) != 1 || result.Parts[0].Text != source {
			t.Fatalf("changed literal Markdown: %+v", result)
		}
	}
}
