package subpages

import (
	_ "embed"
	"html/template"
	"strings"

	"github.com/gi8lino/lore/pluginapi"
)

// renderData contains one shared subpages template invocation.
type renderData struct {
	Children  []pluginapi.NavigationNode
	Title     string
	ShowTitle bool
}

//go:embed template.gohtml
var templateSource string

func newTemplate(icon func(string, int) template.HTML) *template.Template {
	return template.Must(
		template.New("subpages").
			Funcs(template.FuncMap{
				"icon": icon,
			}).
			Parse(templateSource),
	)
}

// NewRenderer returns a renderer for one prepared navigation subtree and URL strategy.
func NewRenderer(nodes []pluginapi.NavigationNode, icon func(string, int) template.HTML) func(Options) (string, error) {
	htmlTemplate := newTemplate(icon)
	return func(options Options) (string, error) {
		if len(nodes) == 0 {
			return "", nil
		}

		var output strings.Builder
		err := htmlTemplate.ExecuteTemplate(&output, "subpage-toc", renderData{
			Children:  nodes,
			Title:     options.Title,
			ShowTitle: options.ShowTitle,
		})
		if err != nil {
			return "", err
		}
		return output.String(), nil
	}
}
