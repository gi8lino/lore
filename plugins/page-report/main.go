package main

import (
	"context"
	"encoding/json"

	"github.com/gi8lino/lore/pluginapi"
	feature "github.com/gi8lino/lore/plugins/features/pagereport"
)

func main() {}

type pages struct{}

func (pages) Search(_ context.Context, query string, limit int) ([]pluginapi.Page, error) {
	var result []pluginapi.Page
	err := pluginapi.Call("pages.search", pluginapi.PageQuery{Query: query, Limit: limit}, &result)
	return result, err
}
func (pages) GetPage(_ context.Context, slug string) (pluginapi.Page, error) {
	var result pluginapi.Page
	err := pluginapi.Call("pages.get", pluginapi.PageRef{Slug: slug}, &result)
	return result, err
}
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
	html, err := feature.NewRenderer(context.Background(), pages{})(options)
	if err != nil {
		return pluginapi.RenderResult{Error: err.Error()}
	}
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: html}}}
}
