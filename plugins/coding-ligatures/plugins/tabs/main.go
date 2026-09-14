package main

import "github.com/gi8lino/lore/pluginapi"

// main runs the package entry point.
func main() {}

// transform converts Material-style tab groups before the core Markdown parser runs.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if request.Stage != "preprocess" {
		return pluginapi.RenderResult{Error: "unsupported stage"}
	}

	return pluginapi.RenderResult{Parts: transformTabs(request.Source)}
}
