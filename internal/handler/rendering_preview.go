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
		"callouts": {
			source:  "!!! tip\nUse a callout for important operational context.\n",
			options: md.Options{Callouts: true},
		},
		"tabs": {
			source:  "=== \"kubectl\"\n\n    `kubectl get pods`\n\n=== \"Helm\"\n\n    `helm list`\n",
			options: md.Options{Tabs: true},
		},
		"details": {
			source:  "??? \"Show command\"\n\n    `kubectl describe pod example`\n",
			options: md.Options{Details: true},
		},
		"table_styles": {
			source: "| Service | Status |\n" +
				"| --- | --- |\n" +
				"| API | Healthy |\n" +
				"| Database | Warning |\n\n" +
				"{table header=blue cell:1,2=green cell:2,2=red}\n",
			options: md.Options{Tables: true, TableStyles: true},
		},
		"table_sorting": {
			source:  "| Service | Replicas |\n| --- | ---: |\n| API | 3 |\n| Database | 1 |\n\n{table sortable}\n",
			options: md.Options{Tables: true, TableSorting: true},
		},
		"table_filtering": {
			source:  "| Service | Owner |\n| --- | --- |\n| API | Platform |\n| Database | Data |\n\n{table filterable}\n",
			options: md.Options{Tables: true, TableFiltering: true},
		},
		"mermaid": {
			source:  "```mermaid\nflowchart LR\n  Markdown --> Lore\n```\n",
			options: md.Options{},
		},
		"tables": {
			source:  "| Command | Purpose |\n| --- | --- |\n| `kubectl get pods` | List pods |\n",
			options: md.Options{Tables: true},
		},
		"strikethrough": {
			source:  `Deploy ~~Friday~~ Monday.`,
			options: md.Options{Strikethrough: true},
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
