package main

import (
	"regexp"

	"github.com/gi8lino/lore/pluginapi"
	"github.com/gi8lino/lore/pluginsdk"
)

// Goldmark's unhighlighted fence output has this shape. Capture its escaped
// source unchanged; the core sanitizer still processes the complete document.
var fence = regexp.MustCompile(`(?s)<pre><code class="language-mermaid">.*?</code></pre>`)

// main runs the package entry point.
func main() {}

func init() { pluginsdk.Register(transform) }

// transform marks Mermaid code blocks for the browser-side renderer.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if request.Stage != "postprocess" {
		return pluginapi.RenderResult{Error: "unsupported stage"}
	}
	output := fence.ReplaceAllString(request.Source, `<div class="lore-plugin-block" data-lore-plugin="io.lore.mermaid" data-lore-module="diagrams">$0</div>`)
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: output}}}
}
