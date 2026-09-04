package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWikiLinksAndCallouts(t *testing.T) {
	t.Parallel()

	renderer := New()
	got, err := renderer.Render("See [[Postgres Restore|the runbook]].\n\n!!! warning\nDanger zone\n")
	require.NoError(t, err)
	assert.Contains(t, got, `href="/pages/postgres-restore"`)
	assert.Contains(t, got, `class="callout warning"`)
}

func TestWikiLinkPrefix(t *testing.T) {
	t.Parallel()

	renderer := New()
	options := DefaultOptions()
	options.WikiLinkPrefix = "/docs/"
	got, err := renderer.RenderResolvedWithOptions("[[Hello World]]", func(target string) string {
		return Slug(target) + "/"
	}, options)
	require.NoError(t, err)
	assert.Contains(t, got, `href="/docs/hello-world/"`)
}

func TestLinksAreUnique(t *testing.T) {
	t.Parallel()

	got := Links("[[Hello World]] [[Hello World]] [[infra/DNS]]")
	assert.Equal(t, []string{"hello-world", "infra/dns"}, got)
}

func TestTabsRenderMarkdownPanels(t *testing.T) {
	t.Parallel()

	renderer := New()
	source := `=== "Linux"

    **apt**

    ` + "```bash" + `
    apt install postgresql
    ` + "```" + `

=== "macOS"

    **brew** install postgresql
`

	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `class="markdown-tabs"`)
	assert.Contains(t, got, `class="markdown-tab active"`)
	assert.Contains(t, got, `Linux`)
	assert.Contains(t, got, `macOS`)
	assert.Contains(t, got, `<strong>apt</strong>`)
	assert.Contains(t, got, `apt install postgresql`)
	assert.Contains(t, got, `<strong>brew</strong> install postgresql`)
}

func TestDetailsRenderMarkdownBody(t *testing.T) {
	t.Parallel()

	renderer := New()
	source := `???+ "Why this works"

    This body contains **Markdown** and [[Another Page]].
`

	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `<details class="markdown-details" open`)
	assert.Contains(t, got, `<summary>Why this works</summary>`)
	assert.Contains(t, got, `<strong>Markdown</strong>`)
	assert.Contains(t, got, `href="/pages/another-page"`)
}

func TestCustomBlocksAreIgnoredInsideFences(t *testing.T) {
	t.Parallel()

	renderer := New()
	source := "```text\n=== \"Not a tab\"\n??? \"Not details\"\n!!! warning\n```\n"

	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.NotContains(t, got, `class="markdown-tabs"`)
	assert.NotContains(t, got, `class="markdown-details"`)
	assert.NotContains(t, got, `class="callout`)
	assert.Contains(t, got, `===`)
	assert.Contains(t, got, `???`)
	assert.Contains(t, got, `!!! warning`)
}

func TestAdditionalCalloutKinds(t *testing.T) {
	t.Parallel()

	renderer := New()
	got, err := renderer.Render("!!! info\nInformation\n\n!!! success\nWorked\n\n!!! danger\nStop\n")
	require.NoError(t, err)
	assert.Contains(t, got, `callout info`)
	assert.Contains(t, got, `callout success`)
	assert.Contains(t, got, `callout danger`)
}

func TestCalloutBodyRendersMarkdown(t *testing.T) {
	t.Parallel()

	renderer := New()
	got, err := renderer.Render("!!! warning\n`$(VAR_NAME)` does not work with **envFrom**!\n")
	require.NoError(t, err)
	assert.Contains(t, got, `<code>$(VAR_NAME)</code>`)
	assert.Contains(t, got, `<strong>envFrom</strong>`)
	assert.NotContains(t, got, "`$(VAR_NAME)`")
}

func TestWikiLinksAreIgnoredInsideFencedCode(t *testing.T) {
	t.Parallel()

	renderer := New()
	source := "Before [[Real Page]]\n\n```text\n[[Literal Link]]\n```\n"

	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `href="/pages/real-page"`)
	assert.NotContains(t, got, `href="/pages/literal-link"`)

	links := Links(source)
	assert.Equal(t, []string{"real-page"}, links)
}

func TestRenderPageExtractsHeadingTextWithoutHTMLMarkup(t *testing.T) {
	t.Parallel()

	renderer := New()
	rendered, err := renderer.RenderPageResolved("# Main *heading*\n\n## Child `code`\n", Slug)
	require.NoError(t, err)
	require.Len(t, rendered.Contents, 2)
	assert.Equal(t, 1, rendered.Contents[0].Level)
	assert.Equal(t, "Main heading", rendered.Contents[0].Title)
	assert.Equal(t, 2, rendered.Contents[1].Level)
	assert.Equal(t, "Child code", rendered.Contents[1].Title)
}

func TestSubpagesFunctionExpandsAtItsMarkdownPosition(t *testing.T) {
	t.Parallel()

	renderer := New()
	rendered, err := renderer.RenderPageResolvedWithFunctions(
		"Before\n\n{{subpages}}\n\nAfter\n",
		Slug,
		DefaultOptions(),
		Functions{Subpages: `<nav class="subpage-toc">Generated pages</nav>`},
	)
	require.NoError(t, err)
	assert.Contains(t, rendered.HTML, "<p>Before</p>\n<nav class=\"subpage-toc\">Generated pages</nav>\n<p>After</p>")
	assert.NotContains(t, rendered.HTML, "{{subpages}}")
}

func TestSubpagesFunctionRemainsLiteralInsideFencedCode(t *testing.T) {
	t.Parallel()

	renderer := New()
	rendered, err := renderer.RenderPageResolvedWithFunctions(
		"```markdown\n{{subpages}}\n```\n",
		Slug,
		DefaultOptions(),
		Functions{Subpages: `<nav>Generated pages</nav>`},
	)
	require.NoError(t, err)
	assert.Contains(t, rendered.HTML, "{{subpages}}")
	assert.NotContains(t, rendered.HTML, "Generated pages")
}

func TestSlugWithoutRegularExpressions(t *testing.T) {
	t.Parallel()

	t.Run("normalizes words", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello-world", Slug("Hello World"))
	})
	t.Run("preserves path separators", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "infra/dns", Slug("infra/DNS"))
	})
	t.Run("collapses whitespace", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "foo-bar", Slug("foo   bar"))
	})
	t.Run("removes punctuation", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "hello-kubernetes", Slug("Hello, Kubernetes!"))
	})
}

func TestTableStyleDirective(t *testing.T) {
	t.Parallel()

	renderer := New()
	source := `| Service | Status | Owner |
| --- | --- | --- |
| API | Healthy | Platform |
| DB | Warning | Data |

{table header=accent col:2=info row:2=warning cell:2,2=danger}
`

	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `lore-table-styled`)
	assert.Contains(t, got, `table-tone-accent`)
	assert.Contains(t, got, `table-tone-info`)
	assert.Contains(t, got, `table-tone-warning`)
	assert.Contains(t, got, `table-tone-danger`)
	assert.NotContains(t, got, `{table`)
}

func TestConfluenceTablePalette(t *testing.T) {
	t.Parallel()

	renderer := New()
	source := `| Service | Status | Owner |
| --- | --- | --- |
| API | Healthy | Platform |
| DB | Warning | Data |

{table header=blue col:1=gray row:1=green cell:2,2=red}
`

	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `table-tone-blue`)
	assert.Contains(t, got, `table-tone-gray`)
	assert.Contains(t, got, `table-tone-green`)
	assert.Contains(t, got, `table-tone-red`)
	assert.NotContains(t, got, `{table`)
}

func TestInteractiveTableDirective(t *testing.T) {
	t.Parallel()

	renderer := New()
	source := `| Service | Replicas |
| --- | ---: |
| API | 3 |
| DB | 1 |

{table sortable filterable}
`

	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `lore-table-sortable`)
	assert.Contains(t, got, `lore-table-filterable`)
	assert.NotContains(t, got, `{table`)
}

func TestConfluenceInteractiveTableDirective(t *testing.T) {
	t.Parallel()

	renderer := New()
	source := `| Column 1 | Column 2 | Column 3 |
| --- | --- | --- |
| | | |
| | | |
| | | |

{table header=gray sortable filterable}
`

	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `class="lore-table-sortable lore-table-filterable lore-table-styled"`)
	assert.Contains(t, got, `class="table-tone-gray"`)
	assert.NotContains(t, got, `{table`)
}

func TestTableDirectiveMarkerUsesNearestPrecedingTable(t *testing.T) {
	t.Parallel()

	rendered := `<div class="table-wrapper"><table><thead><tr><th>Column 1</th></tr></thead><tbody><tr><td>Value</td></tr></tbody></table></div>` +
		`<div class="marker-wrapper"><div class="lore-table-style-marker" data-table-style="{table header=gray sortable filterable}"></div></div>`

	got, err := applyTableDirectiveMarkers(rendered, DefaultOptions())
	require.NoError(t, err)
	assert.Contains(t, got, `class="lore-table-sortable lore-table-filterable lore-table-styled"`)
	assert.Contains(t, got, `class="table-tone-gray"`)
	assert.NotContains(t, got, `lore-table-style-marker`)
}

func TestDisabledInteractiveTableDirectiveRemainsMarkdown(t *testing.T) {
	t.Parallel()

	renderer := New()
	options := DefaultOptions()
	options.TableSorting = false
	options.TableFiltering = false

	source := `| Service | Replicas |
| --- | ---: |
| API | 3 |

{table sortable filterable}
`

	got, err := renderer.RenderResolvedWithOptions(source, Slug, options)
	require.NoError(t, err)
	assert.Contains(t, got, `{table sortable filterable}`)
	assert.NotContains(t, got, `lore-table-sortable`)
	assert.NotContains(t, got, `lore-table-filterable`)
}

func TestTableStyleDirectiveInsideTab(t *testing.T) {
	t.Parallel()

	renderer := New()
	source := `=== "Status"

    | Service | Status |
    | --- | --- |
    | API | Healthy |

    {table header=accent cell:1,2=success}
`

	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `markdown-tabs`)
	assert.Contains(t, got, `table-tone-accent`)
	assert.Contains(t, got, `table-tone-success`)
	assert.NotContains(t, got, `{table`)
}

func TestRenderingOptionsDisableExtensions(t *testing.T) {
	t.Parallel()

	renderer := New()
	options := DefaultOptions()
	options.WikiLinks = false
	options.Callouts = false
	options.TableStyles = false

	source := `[[Runbook]]

!!! warning
Do not restart.

| A | B |
| --- | --- |
| one | two |

{table header=accent}
`
	got, err := renderer.RenderResolvedWithOptions(source, Slug, options)
	require.NoError(t, err)
	assert.NotContains(t, got, `href="/pages/runbook"`)
	assert.NotContains(t, got, `class="callout`)
	assert.Contains(t, got, `{table header=accent}`)
	assert.NotContains(t, got, `table-tone-accent`)
}

func TestOptionalMarkdownExtensions(t *testing.T) {
	t.Parallel()

	renderer := New()
	options := DefaultOptions()
	options.Footnotes = true
	options.DefinitionLists = true

	source := `Lore has a note.[^1]

[^1]: Stored with the page.

Term
: Definition
`
	got, err := renderer.RenderResolvedWithOptions(source, Slug, options)
	require.NoError(t, err)
	assert.Contains(t, got, `footnote`)
	assert.Contains(t, got, `<dl>`)
}

// TestSyntaxHighlightingEmitsChromaClasses verifies highlighted code exposes stable token classes for theme-aware CSS.
func TestSyntaxHighlightingEmitsChromaClasses(t *testing.T) {
	t.Parallel()

	renderer := New()
	options := DefaultOptions()
	options.SyntaxHighlighting = true
	got, err := renderer.RenderResolvedWithOptions("```go\nfunc main() { println(\"Lore\") }\n```\n", Slug, options)
	require.NoError(t, err)
	assert.Contains(t, got, `class="chroma"`)
	assert.Contains(t, got, `class="kd"`)
}
