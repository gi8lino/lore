package main

import "github.com/gi8lino/lore/pluginapi"

// main runs the package entry point.
func main() {}

// transform preserves its input. Coding Ligatures contributes metadata-only
// content-style and render-policy modules, so Lore does not invoke this export.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: request.Source}}}
}
