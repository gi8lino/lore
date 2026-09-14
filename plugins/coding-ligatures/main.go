package main

import (
	"github.com/gi8lino/lore/pluginapi"
	"github.com/gi8lino/lore/pluginsdk"
)

// main runs the package entry point.
func main() {}

func init() { pluginsdk.Register(transform) }

// transform preserves its input. Coding Ligatures contributes metadata-only
// content-style and render-policy modules, so Lore does not invoke this export.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: request.Source}}}
}
