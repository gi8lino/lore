package main

import (
	"html/template"

	"github.com/gi8lino/lore/pluginsdk"
)

func main() {}
func init() { pluginsdk.RegisterMacro("subpages", parse, renderMacro) }
func renderMacro(options macroOptions) (pluginsdk.Result, error) {
	nodes, err := pluginsdk.Pages().Navigation()
	if err != nil {
		return pluginsdk.Result{}, err
	}
	var iconError error
	render := newRenderer(nodes, func(name string, size int) template.HTML {
		value, err := pluginsdk.Icon(name, size)
		if err != nil {
			iconError = err
		}
		return template.HTML(value)
	})
	html, err := render(options)
	if err != nil {
		return pluginsdk.Result{}, err
	}
	if iconError != nil {
		return pluginsdk.Result{}, iconError
	}
	return pluginsdk.Text(html), nil
}
