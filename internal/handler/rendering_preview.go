package handler

import (
	"html/template"

	md "github.com/gi8lino/lore/internal/markdown"
)

// renderingPreview describes one administrator rendering-setting example.
type renderingPreview struct {
	// source is the Markdown rendered for the example.
	source string
	// options enables only the rendering behavior demonstrated by the example.
	options md.Options
}

// renderingPreviews renders compact examples for every administrator-controlled rendering setting.
func renderingPreviews(renderer *md.Renderer) (map[string]template.HTML, error) {
	previews := map[string]renderingPreview{
		"wiki_links": {
			source:  `See [[Keycloak|Keycloak configuration]].`,
			options: md.Options{WikiLinks: true},
		},
		"task_lists": {
			source:  "- [x] Create backup\n- [ ] Run upgrade\n",
			options: md.Options{TaskLists: true},
		},
		"autolinks": {
			source:  `Documentation: https://example.com/docs`,
			options: md.Options{Autolinks: true},
		},
		"syntax_highlighting": {
			source:  "```go\nfunc main() {\n    fmt.Println(\"Lore\")\n}\n```\n",
			options: md.Options{SyntaxHighlighting: true},
		},
		"footnotes": {
			source:  "Lore keeps useful context.[^1]\n\n[^1]: A compact footnote.\n",
			options: md.Options{Footnotes: true},
		},
		"definition_lists": {
			source:  "Runbook\n: A repeatable operational procedure.\n",
			options: md.Options{DefinitionLists: true},
		},
		"typographer": {
			source:  `"Lore" -- documentation...`,
			options: md.Options{Typographer: true},
		},
	}

	result := make(map[string]template.HTML, len(previews))

	for name, preview := range previews {
		rendered, err := renderer.RenderResolvedWithOptions(preview.source, md.Slug, preview.options)
		if err != nil {
			return nil, err
		}

		result[name] = template.HTML(rendered)
	}

	return result, nil
}
