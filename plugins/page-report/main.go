package main

import (
	"context"

	"github.com/gi8lino/lore/pluginapi"
	"github.com/gi8lino/lore/pluginsdk"
)

// main runs the package entry point.
func main() {}

func init() { pluginsdk.RegisterMacro("page-report", parse, renderMacro) }

// pages adapts the Lore page capability to the page-report feature interface.
type pages struct {
}

// Search invokes Lore's page-search host capability.
func (pages) Search(_ context.Context, query string, limit int) ([]pluginapi.Page, error) {
	return pluginsdk.Pages().Search(pluginsdk.PageQuery{Query: query, Limit: limit})
}

// GetPage invokes Lore's page lookup host capability.
func (pages) GetPage(_ context.Context, slug string) (pluginapi.Page, error) {
	return pluginsdk.Pages().Get(slug)
}

// renderMacro renders a parsed report using public page capabilities.
func renderMacro(options macroOptions) (pluginsdk.Result, error) {
	html, err := newRenderer(context.Background(), pages{})(options)
	return pluginsdk.Text(html), err
}
