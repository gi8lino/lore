package main

import (
	"strings"

	"github.com/gi8lino/lore/pluginapi"
	"github.com/gi8lino/lore/pluginsdk"
)

func main() {}

func init() { pluginsdk.RegisterModule("chroma", transform) }

// transform highlights one fenced code block using Lore's explicit fence language.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if request.Module != "chroma" || request.Stage != "highlight" {
		return pluginapi.RenderResult{Error: "unsupported syntax-highlighting request"}
	}

	language, ok := lookupLanguage(strings.TrimSpace(request.Language))
	if !ok {
		return pluginapi.RenderResult{}
	}

	return pluginapi.RenderResult{
		Matched: true,
		Parts: []pluginapi.RenderPart{{
			Text: renderHighlighted(request.Language, request.Source, language),
		}},
	}
}
