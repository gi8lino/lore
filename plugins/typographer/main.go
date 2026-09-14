package main

import (
	"github.com/gi8lino/lore/pluginapi"
	"github.com/gi8lino/lore/pluginsdk"
)

// main runs the package entry point.
func main() {}

func init() { pluginsdk.Register(transform) }

// transform preserves input because this plugin contributes a host rendering policy only.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: request.Source}}}
}
