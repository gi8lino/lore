package markdown

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xhtml "golang.org/x/net/html"
)

func TestImageWidthsRenderInMarkdown(t *testing.T) {
	t.Parallel()

	t.Run("bare pixels", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`![Diagram](diagram.png){width=640}`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:640px", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("explicit pixels", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`![Diagram](diagram.png){width=640px}`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:640px", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("percentage", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`![Diagram](diagram.png){width=50%}`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("linked image", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`[![Diagram](diagram.png){width=50%}](full.png)`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("reference image", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render("![Diagram][image]{width=50%}\n\n[image]: diagram.png")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("collapsed reference", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render("![Diagram][]{width=640}\n\n[Diagram]: diagram.png")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:640px", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("shortcut reference", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render("![Diagram]{width=640}\n\n[Diagram]: diagram.png")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:640px", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("emphasis", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`**![Diagram](diagram.png){width=50%}**`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("blockquote", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`> ![Diagram](diagram.png){width=50%}`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("list", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`- ![Diagram](diagram.png){width=50%}`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("table", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render("| Diagram |\n| --- |\n| ![Diagram](diagram.png){width=50%} |")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("callout", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render("!!! info\n![Diagram](diagram.png){width=50%}\n")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("tab", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render("=== \"Diagram\"\n\n    ![Diagram](diagram.png){width=50%}\n")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})

	t.Run("details", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render("??? \"Diagram\"\n\n    ![Diagram](diagram.png){width=50%}\n")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram", images[0]["alt"])
		assert.NotContains(t, got, "{width=")
	})
}

func TestImageWidthsKeepTitlesURLsAndFollowingText(t *testing.T) {
	t.Parallel()

	got, err := testRenderer(t).Render(`Before ![A & B](images/diagram(v2).png "Overview & details"){width=640} between ![Other](other.png){width=25%} after.`)
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 2)
	assert.Equal(t, "images/diagram(v2).png", images[0]["src"])
	assert.Equal(t, "A & B", images[0]["alt"])
	assert.Equal(t, "Overview & details", images[0]["title"])
	assert.Equal(t, "width:640px", normalizedImageStyle(images[0]["style"]))
	assert.Equal(t, "width:25%", normalizedImageStyle(images[1]["style"]))
	assert.Contains(t, got, "Before ")
	assert.Contains(t, got, " between ")
	assert.Contains(t, got, " after.")
}

func TestAdjacentImageWidths(t *testing.T) {
	t.Parallel()

	got, err := testRenderer(t).Render(`![A](a.png){width=30%}![B](b.png){width=40%}`)
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 2)
	assert.Equal(t, "width:30%", normalizedImageStyle(images[0]["style"]))
	assert.Equal(t, "width:40%", normalizedImageStyle(images[1]["style"]))
}

func TestImageWidthPreservesLineBreaks(t *testing.T) {
	t.Parallel()

	t.Run("soft break", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=50%}\nFollowing")
		require.NoError(t, err)
		assert.Contains(t, strings.ReplaceAll(got, "/>", ">"), ">\nFollowing")
		assert.NotContains(t, got, "{width=")
	})

	t.Run("two spaces", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=50%}  \nFollowing")
		require.NoError(t, err)
		assert.Contains(t, strings.ReplaceAll(got, "/>", ">"), "<br>\nFollowing")
		assert.NotContains(t, got, "{width=")
	})

	t.Run("backslash", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=50%}\\\nFollowing")
		require.NoError(t, err)
		assert.Contains(t, strings.ReplaceAll(got, "/>", ">"), "<br>\nFollowing")
		assert.NotContains(t, got, "{width=")
	})
}

func TestInvalidImageWidthsStayVisible(t *testing.T) {
	t.Parallel()

	t.Run("zero width", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=0}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=0}")
	})

	t.Run("negative width", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=-1}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=-1}")
	})

	t.Run("percentage above maximum", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=101%}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=101%}")
	})

	t.Run("pixels above maximum", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=10001}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=10001}")
	})

	t.Run("fractional percentage", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=50.5%}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50.5%}")
	})

	t.Run("unsupported unit", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=10em}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=10em}")
	})

	t.Run("multiple dimensions", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=50% height=20}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50% height=20}")
	})

	t.Run("extra CSS declaration", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=50%;position:fixed}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%;position:fixed}")
	})

	t.Run("height instead of width", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){height=20}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{height=20}")
	})

	t.Run("unclosed directive", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png){width=50%")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%")
	})
}

func TestImageWidthOnlyConsumesAdjacentUnescapedDirectives(t *testing.T) {
	t.Parallel()

	t.Run("leading space", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png) {width=50%}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("leading newline", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png)\n{width=50%}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("separate paragraph", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png)\n\n{width=50%}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("escaped opening brace", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png)\\{width=50%}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("encoded opening brace", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("![Diagram](diagram.png)&#123;width=50%}")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%}")
	})
}

func TestImageWidthDoesNotInterpretCodeOrOrdinaryLinks(t *testing.T) {
	t.Parallel()

	t.Run("inline code", func(t *testing.T) {
		t.Parallel()
		options := DefaultOptions()
		options.SyntaxHighlighting = false
		got, err := testRenderer(t).RenderResolvedWithOptions("`![Diagram](diagram.png){width=50%}`", Slug, options)
		require.NoError(t, err)
		assert.Empty(t, renderedImageAttributes(t, got))
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("backtick code fence", func(t *testing.T) {
		t.Parallel()
		options := DefaultOptions()
		options.SyntaxHighlighting = false
		got, err := testRenderer(t).RenderResolvedWithOptions("```text\n![Diagram](diagram.png){width=50%}\n```", Slug, options)
		require.NoError(t, err)
		assert.Empty(t, renderedImageAttributes(t, got))
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("tilde code fence", func(t *testing.T) {
		t.Parallel()
		options := DefaultOptions()
		options.SyntaxHighlighting = false
		got, err := testRenderer(t).RenderResolvedWithOptions("~~~text\n![Diagram](diagram.png){width=50%}\n~~~", Slug, options)
		require.NoError(t, err)
		assert.Empty(t, renderedImageAttributes(t, got))
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("indented code", func(t *testing.T) {
		t.Parallel()
		options := DefaultOptions()
		options.SyntaxHighlighting = false
		got, err := testRenderer(t).RenderResolvedWithOptions("    ![Diagram](diagram.png){width=50%}\n", Slug, options)
		require.NoError(t, err)
		assert.Empty(t, renderedImageAttributes(t, got))
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("escaped image", func(t *testing.T) {
		t.Parallel()
		options := DefaultOptions()
		options.SyntaxHighlighting = false
		got, err := testRenderer(t).RenderResolvedWithOptions(`\![Diagram](diagram.png){width=50%}`, Slug, options)
		require.NoError(t, err)
		assert.Empty(t, renderedImageAttributes(t, got))
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("ordinary link", func(t *testing.T) {
		t.Parallel()
		options := DefaultOptions()
		options.SyntaxHighlighting = false
		got, err := testRenderer(t).RenderResolvedWithOptions(`[Diagram](diagram.png){width=50%}`, Slug, options)
		require.NoError(t, err)
		assert.Empty(t, renderedImageAttributes(t, got))
		assert.Contains(t, got, "{width=50%}")
	})
}

func TestImageWidthDoesNotInterpretRawHTMLOrNestedAltText(t *testing.T) {
	t.Parallel()

	t.Run("raw HTML image", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render(`<img src="diagram.png" alt="Diagram">{width=50%}`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("nested image alt text", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render(`![Outer ![Inner](inner.png){width=50%}](outer.png)`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%}")
	})

	t.Run("literal directive in alt text", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render(`![Diagram {width=50%}](diagram.png)`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
		assert.Contains(t, got, "{width=50%}")
	})
}

func TestImageWidthDoesNotRequireOptionalRenderingFeatures(t *testing.T) {
	t.Parallel()

	got, err := testRenderer(t).RenderResolvedWithOptions(`![Diagram](diagram.png){width=50%}`, Slug, Options{})
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 1)
	assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
}

func TestUnsizedImageRenderingIsUnchanged(t *testing.T) {
	t.Parallel()

	got, err := testRenderer(t).Render(`![Diagram](diagram.png "Title")`)
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 1)
	assert.Equal(t, map[string]string{
		"src": "diagram.png", "alt": "Diagram", "title": "Title",
	}, images[0])
}

func TestImageWidthSanitizerAllowsOnlyBoundedWidths(t *testing.T) {
	t.Parallel()

	got, err := testRenderer(t).Render(`<img src="diagram.png" style="width:50%;position:fixed;top:0;height:1px;background:url(https://example.test/track)" onerror="alert(1)"><span style="width:50%">Text</span>`)
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 1)
	assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
	assert.NotContains(t, got, "onerror")
	assert.NotContains(t, got, "position")
	assert.NotContains(t, got, "background")
	assert.NotContains(t, got, "height:")
	assert.Equal(t, 1, strings.Count(got, `style="`), "width styles must not be allowed on spans")

	t.Run("zero pixels", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("<img src=\"diagram.png\" style=\"width:0px\">")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
	})

	t.Run("percentage above maximum", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("<img src=\"diagram.png\" style=\"width:101%\">")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
	})

	t.Run("pixels above maximum", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("<img src=\"diagram.png\" style=\"width:10001px\">")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
	})

	t.Run("CSS expression", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("<img src=\"diagram.png\" style=\"width:expression(alert(1))\">")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
	})

	t.Run("CSS calculation", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("<img src=\"diagram.png\" style=\"width:calc(50% + 1px)\">")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
	})

	t.Run("CSS variable", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("<img src=\"diagram.png\" style=\"width:var(--width)\">")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
	})

	t.Run("CSS URL", func(t *testing.T) {
		t.Parallel()
		got, err := testRenderer(t).Render("<img src=\"diagram.png\" style=\"width:url(https://example.test/track)\">")
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)
		require.Len(t, images, 1)
		assert.Empty(t, images[0]["style"])
	})
}

func TestImageWidthKeepsURLAndAttributeSanitization(t *testing.T) {
	t.Parallel()

	got, err := testRenderer(t).Render(`![Diagram](javascript:alert%281%29){width=50%}`)
	require.NoError(t, err)
	assert.NotContains(t, got, "javascript:")

	got, err = testRenderer(t).Render(`![Diagram](diagram.png){width=50% onerror="alert(1)"}`)
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 1)
	assert.Empty(t, images[0]["style"])
	assert.Empty(t, images[0]["onerror"])
	assert.Contains(t, got, "{width=")
}

// renderedImageAttributes reads actual image attributes rather than text or code examples.
func renderedImageAttributes(t *testing.T, rendered string) []map[string]string {
	t.Helper()
	var images []map[string]string
	tokens := xhtml.NewTokenizer(strings.NewReader(rendered))
	for {
		kind := tokens.Next()
		if kind == xhtml.ErrorToken {
			require.ErrorIs(t, tokens.Err(), io.EOF)
			return images
		}
		if kind != xhtml.StartTagToken && kind != xhtml.SelfClosingTagToken {
			continue
		}
		token := tokens.Token()
		if token.Data != "img" {
			continue
		}
		attributes := make(map[string]string, len(token.Attr))
		for _, attribute := range token.Attr {
			attributes[attribute.Key] = attribute.Val
		}
		images = append(images, attributes)
	}
}

func normalizedImageStyle(value string) string {
	return strings.TrimSuffix(strings.ReplaceAll(value, " ", ""), ";")
}

func TestImageTextAttributesSurviveSanitization(t *testing.T) {
	t.Parallel()

	t.Run("ampersands", func(t *testing.T) {
		t.Parallel()

		// Isolate the sanitizer from Goldmark to catch restrictive attribute
		// policies even when the Markdown parser already emits correct HTML.
		source := `<img src="diagram.png" alt="` + html.EscapeString("A & B") +
			`" title="` + html.EscapeString("Overview & details") + `">`
		got := testRenderer(t).sanitizer.Sanitize(source)
		images := renderedImageAttributes(t, got)

		require.Len(t, images, 1)
		assert.Equal(t, map[string]string{
			"src": "diagram.png", "alt": "A & B", "title": "Overview & details",
		}, images[0])
	})

	t.Run("literal width syntax", func(t *testing.T) {
		t.Parallel()

		// Isolate the sanitizer from Goldmark to catch restrictive attribute
		// policies even when the Markdown parser already emits correct HTML.
		source := `<img src="diagram.png" alt="` + html.EscapeString("Outer Inner{width=50%}") +
			`" title="` + html.EscapeString("Example {width=640}") + `">`
		got := testRenderer(t).sanitizer.Sanitize(source)
		images := renderedImageAttributes(t, got)

		require.Len(t, images, 1)
		assert.Equal(t, map[string]string{
			"src": "diagram.png", "alt": "Outer Inner{width=50%}", "title": "Example {width=640}",
		}, images[0])
	})

	t.Run("punctuation", func(t *testing.T) {
		t.Parallel()

		// Isolate the sanitizer from Goldmark to catch restrictive attribute
		// policies even when the Markdown parser already emits correct HTML.
		source := `<img src="diagram.png" alt="` + html.EscapeString("A: B; C? D=E | F + G @ H") +
			`" title="` + html.EscapeString("50% / $20 #1 ~ result") + `">`
		got := testRenderer(t).sanitizer.Sanitize(source)
		images := renderedImageAttributes(t, got)

		require.Len(t, images, 1)
		assert.Equal(t, map[string]string{
			"src": "diagram.png", "alt": "A: B; C? D=E | F + G @ H", "title": "50% / $20 #1 ~ result",
		}, images[0])
	})

	t.Run("quotes and angle brackets", func(t *testing.T) {
		t.Parallel()

		// Isolate the sanitizer from Goldmark to catch restrictive attribute
		// policies even when the Markdown parser already emits correct HTML.
		source := `<img src="diagram.png" alt="` + html.EscapeString(`"Quoted" <diagram> & 'label'`) +
			`" title="` + html.EscapeString(`<title> "quoted" & 'single'`) + `">`
		got := testRenderer(t).sanitizer.Sanitize(source)
		images := renderedImageAttributes(t, got)

		require.Len(t, images, 1)
		assert.Equal(t, map[string]string{
			"src": "diagram.png", "alt": `"Quoted" <diagram> & 'label'`, "title": `<title> "quoted" & 'single'`,
		}, images[0])
	})

	t.Run("unicode", func(t *testing.T) {
		t.Parallel()

		// Isolate the sanitizer from Goldmark to catch restrictive attribute
		// policies even when the Markdown parser already emits correct HTML.
		source := `<img src="diagram.png" alt="` + html.EscapeString("Gr\u00f6sse \u2192 50% \U0001f4f7") +
			`" title="` + html.EscapeString("\u6982\u8981 & \u8a73\u7d30") + `">`
		got := testRenderer(t).sanitizer.Sanitize(source)
		images := renderedImageAttributes(t, got)

		require.Len(t, images, 1)
		assert.Equal(t, map[string]string{
			"src": "diagram.png", "alt": "Gr\u00f6sse \u2192 50% \U0001f4f7", "title": "\u6982\u8981 & \u8a73\u7d30",
		}, images[0])
	})

	t.Run("empty text", func(t *testing.T) {
		t.Parallel()

		// Isolate the sanitizer from Goldmark to catch restrictive attribute
		// policies even when the Markdown parser already emits correct HTML.
		source := `<img src="diagram.png" alt="` + html.EscapeString("") +
			`" title="` + html.EscapeString("") + `">`
		got := testRenderer(t).sanitizer.Sanitize(source)
		images := renderedImageAttributes(t, got)

		require.Len(t, images, 1)
		assert.Equal(t, map[string]string{
			"src": "diagram.png", "alt": "", "title": "",
		}, images[0])
	})
}

func TestImageTextAttributesWithAndWithoutWidths(t *testing.T) {
	t.Parallel()

	t.Run("unsized image", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`![A & B](diagram.png "Overview & details")`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)

		require.Len(t, images, 1)
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "A & B", images[0]["alt"])
		assert.Equal(t, "Overview & details", images[0]["title"])
		assert.Equal(t, "", normalizedImageStyle(images[0]["style"]))
	})

	t.Run("sized image", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`![A & B](diagram.png "Overview & details"){width=640}`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)

		require.Len(t, images, 1)
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "A & B", images[0]["alt"])
		assert.Equal(t, "Overview & details", images[0]["title"])
		assert.Equal(t, "width:640px", normalizedImageStyle(images[0]["style"]))
	})

	t.Run("width-like alt text", func(t *testing.T) {
		t.Parallel()

		got, err := testRenderer(t).Render(`![Diagram {width=50%}](diagram.png "Overview & details"){width=640}`)
		require.NoError(t, err)
		images := renderedImageAttributes(t, got)

		require.Len(t, images, 1)
		assert.Equal(t, "diagram.png", images[0]["src"])
		assert.Equal(t, "Diagram {width=50%}", images[0]["alt"])
		assert.Equal(t, "Overview & details", images[0]["title"])
		assert.Equal(t, "width:640px", normalizedImageStyle(images[0]["style"]))
	})
}

func TestImageTextAttributesCannotInjectMarkup(t *testing.T) {
	t.Parallel()

	const alt = `"><script>alert(1)</script><img src=x onerror=alert(2)>`
	const title = `" onmouseover="alert(3)" & {width=50%}`
	source := `<img src="diagram.png" alt="` + html.EscapeString(alt) +
		`" title="` + html.EscapeString(title) +
		`" onerror="alert(4)" onload="alert(5)" style="width:50%;position:fixed">` +
		`<script>alert(6)</script>`

	got, err := testRenderer(t).RenderResolvedWithOptions(source, Slug, Options{})
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)

	require.Len(t, images, 1)
	assert.Equal(t, alt, images[0]["alt"])
	assert.Equal(t, title, images[0]["title"])
	assert.Equal(t, "diagram.png", images[0]["src"])
	assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
	assert.Len(t, images[0], 4, "only src, alt, title and the validated style may remain")
	assert.NotContains(t, got, "<script")
	assert.NotContains(t, got, "alert(6)")
	assert.NotContains(t, got, "position:")
}

func TestNormalizeImageWidth(t *testing.T) {
	t.Parallel()

	t.Run("minimum pixels", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "1px", normalizeImageWidth("1"))
	})

	t.Run("bare pixels", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "640px", normalizeImageWidth("640"))
	})

	t.Run("explicit pixels", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "640px", normalizeImageWidth("640px"))
	})

	t.Run("leading zeroes", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "640px", normalizeImageWidth("000640"))
	})

	t.Run("maximum pixels", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "10000px", normalizeImageWidth("10000px"))
	})

	t.Run("minimum percentage", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "1%", normalizeImageWidth("1%"))
	})

	t.Run("percentage", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "50%", normalizeImageWidth("50%"))
	})

	t.Run("maximum percentage", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "100%", normalizeImageWidth("100%"))
	})

	t.Run("empty width", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth(""))
	})

	t.Run("zero bare width", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("0"))
	})

	t.Run("zero pixels", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("0px"))
	})

	t.Run("zero percentage", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("0%"))
	})

	t.Run("negative width", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("-1"))
	})

	t.Run("explicit positive sign", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("+1"))
	})

	t.Run("pixels above maximum", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("10001px"))
	})

	t.Run("percentage above maximum", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("101%"))
	})

	t.Run("fractional percentage", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("50.5%"))
	})

	t.Run("fractional pixels", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("1.5px"))
	})

	t.Run("scientific notation", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("1e2"))
	})

	t.Run("hexadecimal", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("0x10"))
	})

	t.Run("duplicate percentage unit", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("50%%"))
	})

	t.Run("percentage after pixels", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("50px%"))
	})

	t.Run("pixels after percentage", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("50%px"))
	})

	t.Run("em units", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("10em"))
	})

	t.Run("viewport units", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("10vw"))
	})

	t.Run("uppercase pixel unit", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("640PX"))
	})

	t.Run("automatic width", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("auto"))
	})

	t.Run("leading space", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth(" 640"))
	})

	t.Run("trailing space", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("640 "))
	})

	t.Run("space before unit", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("640 px"))
	})

	t.Run("trailing newline", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("640\n"))
	})

	t.Run("null byte", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("640\u0000"))
	})

	t.Run("full-width digits", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("\uff16\uff14\uff10"))
	})

	t.Run("integer overflow", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("999999999999999999999999999999999"))
	})

	t.Run("extra CSS declaration", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("50%;position:fixed"))
	})

	t.Run("CSS calculation", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("calc(100% - 1px)"))
	})

	t.Run("CSS URL", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("url(https://example.test/image)"))
	})

	t.Run("CSS variable", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, normalizeImageWidth("var(--width)"))
	})
}

func TestImageWidthRanges(t *testing.T) {
	t.Parallel()

	t.Run("normalizes every supported pixel width", func(t *testing.T) {
		t.Parallel()

		for width := 1; width <= maxImageWidthPixels; width++ {
			value := strconv.Itoa(width)
			require.Equal(t, value+"px", normalizeImageWidth(value), "pixel width %d", width)
		}
	})

	t.Run("sanitizer accepts every supported pixel width", func(t *testing.T) {
		t.Parallel()

		for width := 1; width <= maxImageWidthPixels; width++ {
			value := strconv.Itoa(width)
			require.True(t, validImageWidthStyle(value+"px"), "pixel width %d", width)
		}
	})

	t.Run("sanitizer limits percentages to one hundred", func(t *testing.T) {
		t.Parallel()

		for width := 1; width <= maxImageWidthPixels; width++ {
			value := strconv.Itoa(width)
			require.Equal(t, width <= 100, validImageWidthStyle(value+"%"), "percentage %d", width)
		}
	})
}

func TestParseImageWidthDirective(t *testing.T) {
	t.Parallel()

	t.Run("bare pixels", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=640}"))

		assert.Equal(t, "640px", width)
		assert.Equal(t, len("{width=640}"), consumed)
	})

	t.Run("pixels followed by text", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=640px} after"))

		assert.Equal(t, "640px", width)
		assert.Equal(t, len("{width=640px}"), consumed)
	})

	t.Run("percentage followed by image", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=50%}![next](b.png)"))

		assert.Equal(t, "50%", width)
		assert.Equal(t, len("{width=50%}"), consumed)
	})

	t.Run("percentage followed by newline", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=50%}\nNext line"))

		assert.Equal(t, "50%", width)
		assert.Equal(t, len("{width=50%}"), consumed)
	})

	t.Run("consumes only the first directive", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=50%}{width=25%}"))

		assert.Equal(t, "50%", width)
		assert.Equal(t, len("{width=50%}"), consumed)
	})

	t.Run("empty source", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte(""))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("missing value", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width="))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("unclosed directive", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=50%"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("zero width", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=0}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("percentage above maximum", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=101%}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("leading space", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte(" {width=50%}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("leading newline", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("\n{width=50%}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("escaped opening brace", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("\\{width=50%}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("encoded opening brace", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("&#123;width=50%}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("space before equals", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width =50%}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("space after width", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=50% }"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("height directive", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{height=50}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("multiple dimensions", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=50% height=20}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})

	t.Run("extra CSS declaration", func(t *testing.T) {
		t.Parallel()

		width, consumed := parseImageWidthDirective([]byte("{width=50%;position:fixed}"))

		assert.Empty(t, width)
		assert.Zero(t, consumed)
	})
}

func TestValidImageWidthStyle(t *testing.T) {
	t.Parallel()

	t.Run("empty style", func(t *testing.T) {
		t.Parallel()

		assert.False(t, validImageWidthStyle(""))
	})

	t.Run("unitless width", func(t *testing.T) {
		t.Parallel()

		assert.False(t, validImageWidthStyle("640"))
	})

	t.Run("leading zeroes", func(t *testing.T) {
		t.Parallel()

		assert.False(t, validImageWidthStyle("000640px"))
	})

	t.Run("zero pixels", func(t *testing.T) {
		t.Parallel()

		assert.False(t, validImageWidthStyle("0px"))
	})

	t.Run("percentage above maximum", func(t *testing.T) {
		t.Parallel()

		assert.False(t, validImageWidthStyle("101%"))
	})

	t.Run("pixels above maximum", func(t *testing.T) {
		t.Parallel()

		assert.False(t, validImageWidthStyle("10001px"))
	})

	t.Run("extra CSS declaration", func(t *testing.T) {
		t.Parallel()

		assert.False(t, validImageWidthStyle("50%;position:fixed"))
	})

	t.Run("important modifier", func(t *testing.T) {
		t.Parallel()

		assert.False(t, validImageWidthStyle("50% !important"))
	})
}

func FuzzParseImageWidthDirective(f *testing.F) {
	f.Add("")
	f.Add("{width=640}")
	f.Add("{width=50%} after")
	f.Add("{width=0}")
	f.Add("{width=-1}")
	f.Add("{width=50%;position:fixed}")

	f.Fuzz(func(t *testing.T, source string) {
		width, consumed := parseImageWidthDirective([]byte(source))
		if consumed == 0 {
			assert.Empty(t, width, "rejected directives must not return a width")
			return
		}

		require.Positive(t, consumed)
		require.LessOrEqual(t, consumed, len(source))
		assert.Equal(t, byte('}'), source[consumed-1])
		assert.True(t, validImageWidthStyle(width), "parsed width %q", width)
	})
}
