package main

import (
	"encoding/json"
	"html/template"

	"github.com/gi8lino/lore/pluginapi"
	feature "github.com/gi8lino/lore/plugins/features/subpages"
)

func main() {}
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if request.Stage == "parse" {
		options, matched := feature.Parse(request.Source)
		data, _ := json.Marshal(options)
		return pluginapi.RenderResult{Matched: matched, Invocation: data}
	}
	var options feature.Options
	if err := json.Unmarshal(request.Invocation, &options); err != nil {
		return pluginapi.RenderResult{Error: err.Error()}
	}
	var nodes []pluginapi.NavigationNode
	if err := pluginapi.Call("pages.navigation", nil, &nodes); err != nil {
		return pluginapi.RenderResult{Error: err.Error()}
	}
	var iconError error
	render := feature.NewRenderer(nodes, func(name string, size int) template.HTML {
		var value string
		if err := pluginapi.Call("icons.render", pluginapi.IconRequest{Name: name, Size: size}, &value); err != nil {
			iconError = err
		}
		return template.HTML(value)
	})
	html, err := render(options)
	if err != nil {
		return pluginapi.RenderResult{Error: err.Error()}
	}
	if iconError != nil {
		return pluginapi.RenderResult{Error: iconError.Error()}
	}
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: html}}}
}
