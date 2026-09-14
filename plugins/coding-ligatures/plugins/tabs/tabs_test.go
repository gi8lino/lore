package main

import (
	"strings"
	"testing"

	"github.com/gi8lino/lore/pluginapi"
)

func TestTransformTabs(t *testing.T) {
	result := transform(pluginapi.RenderRequest{Stage: "preprocess", Source: "Before\n\n=== \"Linux\"\n\n    **apt**\n\n=== \"macOS\"\n\n    `brew`\n\nAfter"})
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

	if !strings.Contains(text.String(), `class="markdown-tabs"`) || !strings.Contains(text.String(), `>Linux</button>`) {
		t.Fatalf("tab markup missing: %s", text.String())
	}
	if len(markdown) != 2 || !strings.Contains(markdown[0], "**apt**") || !strings.Contains(markdown[1], "`brew`") {
		t.Fatalf("unexpected panel markdown: %#v", markdown)
	}
}

func TestTransformTabsIgnoresFences(t *testing.T) {
	source := "```text\n=== \"Not a tab\"\n```\n"
	result := transform(pluginapi.RenderRequest{Stage: "preprocess", Source: source})
	if len(result.Parts) != 1 || result.Parts[0].Text != source {
		t.Fatalf("fenced source changed: %#v", result.Parts)
	}
}
