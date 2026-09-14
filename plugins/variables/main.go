package main

import "github.com/gi8lino/lore/pluginapi"

// main runs the package entry point.
func main() {}

// transform preserves input because Variables uses host-provided resource modules only.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: request.Source}}}
}
