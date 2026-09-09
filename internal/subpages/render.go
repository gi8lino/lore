package subpages

import (
	_ "embed"
	"html/template"
	"strings"

	"github.com/gi8lino/lore/internal/icons"
	"github.com/gi8lino/lore/internal/navigation"
)

// renderData contains one shared subpages template invocation.
type renderData struct {
	Children  []renderNode
	Title     string
	ShowTitle bool
}

// renderNode contains presentation data for one navigation node.
type renderNode struct {
	Title    string
	Icon     string
	URL      string
	Page     bool
	Children []renderNode
}

//go:embed template.gohtml
var templateSource string

var htmlTemplate = template.Must(
	template.New("subpages").
		Funcs(template.FuncMap{
			"icon": icons.SVG,
		}).
		Parse(templateSource),
)

// NewRenderer returns a renderer for one prepared navigation subtree and URL strategy.
func NewRenderer(children []navigation.Node, pageURL func(string) string) func(Options) (string, error) {
	nodes := renderNodes(children, pageURL)
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

// renderNodes converts navigation nodes into presentation data with resolved page URLs.
func renderNodes(nodes []navigation.Node, pageURL func(string) string) []renderNode {
	result := make([]renderNode, 0, len(nodes))
	for _, node := range nodes {
		rendered := renderNode{
			Title:    node.Title,
			Icon:     node.Icon,
			Page:     node.Page,
			Children: renderNodes(node.Children, pageURL),
		}
		if node.Page {
			rendered.URL = pageURL(node.Slug)
		}
		result = append(result, rendered)
	}
	return result
}
