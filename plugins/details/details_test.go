package main

import (
	"strings"
	"testing"

	"github.com/gi8lino/lore/pluginapi"
)

func TestTransformDetails(t *testing.T) {
	result := transform(pluginapi.RenderRequest{Stage: "preprocess", Source: "Before\n\n???+ \"Show command\"\n\n    **Markdown**\n\nAfter"})
	if result.Error != "" {
		t.Fatal(result.Error)
	}

	var text strings.Builder
	var markdown []string
	for _, part := range result.Parts {
		text.WriteString(part.Text)
		if part.Markdown != nil {
			markdown = append(markdown, *part.Markdown)
		}
	}

	if !strings.Contains(text.String(), `<details class="markdown-details" open>`) || !strings.Contains(text.String(), `<summary>Show command</summary>`) {
		t.Fatalf("details markup missing: %s", text.String())
	}
	if len(markdown) != 1 || !strings.Contains(markdown[0], "**Markdown**") {
		t.Fatalf("unexpected details markdown: %#v", markdown)
	}
}

func TestTransformDetailsIgnoresFences(t *testing.T) {
	source := "```text\n??? \"Not details\"\n```\n"
	result := transform(pluginapi.RenderRequest{Stage: "preprocess", Source: source})
	if len(result.Parts) != 1 || result.Parts[0].Text != source {
		t.Fatalf("fenced source changed: %#v", result.Parts)
	}
}
