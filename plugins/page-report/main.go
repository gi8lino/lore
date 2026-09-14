package main

import (
	"context"
	"encoding/json"

	"github.com/gi8lino/lore/pluginapi"
)

// main runs the package entry point.
func main() {}

// pages adapts the Lore page capability to the page-report feature interface.
type pages struct {
}

// Search invokes Lore's page-search host capability.
func (pages) Search(_ context.Context, query string, limit int) ([]pluginapi.Page, error) {
	var result []pluginapi.Page
	err := pluginapi.Call("pages.search", pluginapi.PageQuery{Query: query, Limit: limit}, &result)
	return result, err
}

// GetPage invokes Lore's page lookup host capability.
func (pages) GetPage(_ context.Context, slug string) (pluginapi.Page, error) {
	var result pluginapi.Page
	err := pluginapi.Call("pages.get", pluginapi.PageRef{Slug: slug}, &result)
	return result, err
}

// transform parses or renders one page-report macro request.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if request.Stage == "parse" {
		options, matched := Parse(request.Source)
		data, _ := json.Marshal(options)
		return pluginapi.RenderResult{Matched: matched, Invocation: data}
	}
	var options Options
	if err := json.Unmarshal(request.Invocation, &options); err != nil {
		return pluginapi.RenderResult{Error: err.Error()}
	}
	html, err := NewRenderer(context.Background(), pages{})(options)
	if err != nil {
		return pluginapi.RenderResult{Error: err.Error()}
	}
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: html}}}
}
