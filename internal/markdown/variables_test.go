package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const testVariableToken = "lorevariable0123456789abcdef0123456789abcdefn0end"

func renderVariableTestPage(t *testing.T, source, value string) RenderedPage {
	t.Helper()
	page, err := New().RenderPageResolvedWithFunctions(source, Slug, DefaultOptions(), Functions{
		Variables: []Variable{{Token: testVariableToken, Name: "environment", Value: value}},
	})
	require.NoError(t, err)
	assert.NotContains(t, page.HTML, testVariableToken)
	return page
}

func TestVariableAnnotations(t *testing.T) {
	t.Parallel()
	t.Run("annotates only macro-origin text", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, "production "+testVariableToken+" production", "production")
		assert.Contains(t, page.HTML, `production <span class="page-variable" data-page-variable="environment">production</span> production`)
		assert.Equal(t, 1, strings.Count(page.HTML, `data-page-variable=`))
	})
	t.Run("preserves real heading identifiers and contents", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, "## Deploy "+testVariableToken, "production")
		require.Len(t, page.Contents, 1)
		assert.Equal(t, "deploy-production", page.Contents[0].ID)
		assert.Equal(t, "Deploy production", page.Contents[0].Title)
		assert.Contains(t, page.HTML, `data-page-variable="environment"`)
	})
	t.Run("annotates a value inside inline code without interpreting HTML", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, "`echo "+testVariableToken+" production`", "staging")
		assert.Contains(t, page.HTML, `<code>echo <span class="page-variable" data-page-variable="environment">staging</span> production</code>`)
	})
	t.Run("keeps image alt text and width attributes", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, "!["+testVariableToken+"](image.png){width=50%}\n\n"+testVariableToken, "A & B")
		assert.Contains(t, page.HTML, `alt="A &amp; B"`)
		assert.Contains(t, page.HTML, `style="width: 50%"`)
		assert.Contains(t, page.HTML, `data-page-variable="environment"`)
	})
	t.Run("preserves link destinations and ordinary equal text", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, "[Visit](https://"+testVariableToken+"/docs) host.example "+testVariableToken, "host.example")
		assert.Contains(t, page.HTML, `href="https://host.example/docs"`)
		assert.Equal(t, "host.example", annotatedVariableText(t, page.HTML))
	})
	t.Run("preserves sanitized Markdown in saved values", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, "Deploy "+testVariableToken, "**production**")
		assert.Contains(t, page.HTML, `<strong><span class="page-variable" data-page-variable="environment">production</span></strong>`)
	})
	t.Run("uses the normal render for complex block values", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, testVariableToken, "!!! warning\nImportant")
		assert.Contains(t, page.HTML, `class="callout warning"`)
		assert.Contains(t, page.HTML, "Important")
	})
	t.Run("leaves fenced macro examples untouched", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, "```text\n{{var:environment}}\n```\n\n"+testVariableToken, "production")
		assert.Contains(t, page.HTML, "{{var:environment}}")
		assert.Equal(t, 1, strings.Count(page.HTML, `data-page-variable=`))
	})
	t.Run("does not permit script or event-handler injection", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, testVariableToken, `<script>alert(1)</script><img src="x" onerror="alert(1)">`)
		assert.NotContains(t, page.HTML, "<script")
		assert.NotContains(t, page.HTML, "onerror")
	})
	t.Run("escapes variable names as attribute text", func(t *testing.T) {
		t.Parallel()
		page, err := New().RenderPageResolvedWithFunctions(testVariableToken, Slug, DefaultOptions(), Functions{Variables: []Variable{
			{Token: testVariableToken, Name: `name" onclick="attack`, Value: "value"},
		}})
		require.NoError(t, err)
		assert.NotContains(t, page.HTML, ` onclick="attack`)
		assert.Contains(t, page.HTML, "value")
	})
	t.Run("restores an empty value without leaking tokens", func(t *testing.T) {
		t.Parallel()
		page := renderVariableTestPage(t, "before "+testVariableToken+" after", "")
		assert.Equal(t, "<p>before  after</p>\n", page.HTML)
	})
}

func annotatedVariableText(t *testing.T, value string) string {
	t.Helper()

	root := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := xhtml.ParseFragment(strings.NewReader(value), root)
	require.NoError(t, err)

	for _, node := range nodes {
		root.AppendChild(node)
	}

	var output strings.Builder
	var appendText func(*xhtml.Node)
	appendText = func(node *xhtml.Node) {
		if node.Type == xhtml.TextNode {
			output.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			appendText(child)
		}
	}

	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode && node.Data == "span" && htmlAttribute(node, "data-page-variable") != "" {
			appendText(node)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(root)
	return output.String()
}

func TestResolveVariableTokens(t *testing.T) {
	t.Parallel()
	t.Run("tracks UTF-8 byte offsets", func(t *testing.T) {
		t.Parallel()
		value, regions := resolveVariableTokens("\u00e4 "+testVariableToken, []Variable{{Token: testVariableToken, Name: "name", Value: "\u6771\u4eac"}})
		assert.Equal(t, "\u00e4 \u6771\u4eac", value)
		require.Len(t, regions, 1)
		assert.Equal(t, 3, regions[0].start)
		assert.Equal(t, 9, regions[0].end)
	})
	t.Run("does not recursively expand replacement tokens", func(t *testing.T) {
		t.Parallel()
		value, regions := resolveVariableTokens(testVariableToken, []Variable{{Token: testVariableToken, Name: "name", Value: testVariableToken}})
		assert.Equal(t, testVariableToken, value)
		assert.Len(t, regions, 1)
	})
	t.Run("does not interpret an unregistered token", func(t *testing.T) {
		t.Parallel()
		value, regions := resolveVariableTokens(testVariableToken, []Variable{{Token: "other", Name: "name", Value: "replacement"}})
		assert.Equal(t, testVariableToken, value)
		assert.Empty(t, regions)
	})
}
