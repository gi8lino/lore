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

	for _, test := range []struct {
		name, source, width string
	}{
		{"bare pixels", `![Diagram](diagram.png){width=640}`, "640px"},
		{"explicit pixels", `![Diagram](diagram.png){width=640px}`, "640px"},
		{"percentage", `![Diagram](diagram.png){width=50%}`, "50%"},
		{"linked image", `[![Diagram](diagram.png){width=50%}](full.png)`, "50%"},
		{"reference image", "![Diagram][image]{width=50%}\n\n[image]: diagram.png", "50%"},
		{"collapsed reference", "![Diagram][]{width=640}\n\n[Diagram]: diagram.png", "640px"},
		{"shortcut reference", "![Diagram]{width=640}\n\n[Diagram]: diagram.png", "640px"},
		{"emphasis", `**![Diagram](diagram.png){width=50%}**`, "50%"},
		{"blockquote", `> ![Diagram](diagram.png){width=50%}`, "50%"},
		{"list", `- ![Diagram](diagram.png){width=50%}`, "50%"},
		{"table", "| Diagram |\n| --- |\n| ![Diagram](diagram.png){width=50%} |", "50%"},
		{"callout", "!!! info\n![Diagram](diagram.png){width=50%}\n", "50%"},
		{"tab", "=== \"Diagram\"\n\n    ![Diagram](diagram.png){width=50%}\n", "50%"},
		{"details", "??? \"Diagram\"\n\n    ![Diagram](diagram.png){width=50%}\n", "50%"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := New().Render(test.source)
			require.NoError(t, err)
			images := renderedImageAttributes(t, got)
			require.Len(t, images, 1)
			assert.Equal(t, "width:"+test.width, normalizedImageStyle(images[0]["style"]))
			assert.Equal(t, "diagram.png", images[0]["src"])
			assert.Equal(t, "Diagram", images[0]["alt"])
			assert.NotContains(t, got, "{width=")
		})
	}
}

func TestImageWidthsKeepTitlesURLsAndFollowingText(t *testing.T) {
	t.Parallel()

	got, err := New().Render(`Before ![A & B](images/diagram(v2).png "Overview & details"){width=640} between ![Other](other.png){width=25%} after.`)
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

	got, err := New().Render(`![A](a.png){width=30%}![B](b.png){width=40%}`)
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 2)
	assert.Equal(t, "width:30%", normalizedImageStyle(images[0]["style"]))
	assert.Equal(t, "width:40%", normalizedImageStyle(images[1]["style"]))
}

func TestImageWidthPreservesLineBreaks(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, separator, want string }{
		{"soft break", "\n", ">\nFollowing"},
		{"two spaces", "  \n", "<br>\nFollowing"},
		{"backslash", "\\\n", "<br>\nFollowing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := New().Render("![Diagram](diagram.png){width=50%}" + test.separator + "Following")
			require.NoError(t, err)
			assert.Contains(t, strings.ReplaceAll(got, "/>", ">"), test.want)
			assert.NotContains(t, got, "{width=")
		})
	}
}

func TestInvalidImageWidthsStayVisible(t *testing.T) {
	t.Parallel()

	for _, directive := range []string{
		"{width=0}", "{width=-1}", "{width=101%}", "{width=10001}",
		"{width=50.5%}", "{width=10em}", "{width=50% height=20}",
		"{width=50%;position:fixed}", "{height=20}", "{width=50%",
	} {
		t.Run(directive, func(t *testing.T) {
			t.Parallel()
			got, err := New().Render("![Diagram](diagram.png)" + directive)
			require.NoError(t, err)
			images := renderedImageAttributes(t, got)
			require.Len(t, images, 1)
			assert.Empty(t, images[0]["style"])
			assert.Contains(t, got, directive)
		})
	}
}

func TestImageWidthOnlyConsumesAdjacentUnescapedDirectives(t *testing.T) {
	t.Parallel()

	for _, suffix := range []string{" {width=50%}", "\n{width=50%}", "\n\n{width=50%}", `\{width=50%}`, "&#123;width=50%}"} {
		t.Run(suffix, func(t *testing.T) {
			t.Parallel()
			got, err := New().Render("![Diagram](diagram.png)" + suffix)
			require.NoError(t, err)
			images := renderedImageAttributes(t, got)
			require.Len(t, images, 1)
			assert.Empty(t, images[0]["style"])
			assert.Contains(t, got, "{width=50%}")
		})
	}
}

func TestImageWidthDoesNotInterpretCodeOrOrdinaryLinks(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		"`![Diagram](diagram.png){width=50%}`",
		"```text\n![Diagram](diagram.png){width=50%}\n```",
		"~~~text\n![Diagram](diagram.png){width=50%}\n~~~",
		"    ![Diagram](diagram.png){width=50%}\n",
		`\![Diagram](diagram.png){width=50%}`,
		`[Diagram](diagram.png){width=50%}`,
	} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			options := DefaultOptions()
			options.SyntaxHighlighting = false
			got, err := New().RenderResolvedWithOptions(source, Slug, options)
			require.NoError(t, err)
			assert.Empty(t, renderedImageAttributes(t, got))
			assert.Contains(t, got, "{width=50%}")
		})
	}
}

func TestImageWidthDoesNotInterpretRawHTMLOrNestedAltText(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		`<img src="diagram.png" alt="Diagram">{width=50%}`,
		`![Outer ![Inner](inner.png){width=50%}](outer.png)`,
		`![Diagram {width=50%}](diagram.png)`,
	} {
		t.Run(source, func(t *testing.T) {
			t.Parallel()
			got, err := New().Render(source)
			require.NoError(t, err)
			images := renderedImageAttributes(t, got)
			require.Len(t, images, 1)
			assert.Empty(t, images[0]["style"])
			assert.Contains(t, got, "{width=50%}")
		})
	}
}

func TestImageWidthDoesNotRequireOptionalRenderingFeatures(t *testing.T) {
	t.Parallel()

	got, err := New().RenderResolvedWithOptions(`![Diagram](diagram.png){width=50%}`, Slug, Options{})
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 1)
	assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
}

func TestUnsizedImageRenderingIsUnchanged(t *testing.T) {
	t.Parallel()

	got, err := New().Render(`![Diagram](diagram.png "Title")`)
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 1)
	assert.Equal(t, map[string]string{
		"src": "diagram.png", "alt": "Diagram", "title": "Title",
	}, images[0])
}

func TestImageWidthSanitizerAllowsOnlyBoundedWidths(t *testing.T) {
	t.Parallel()

	got, err := New().Render(`<img src="diagram.png" style="width:50%;position:fixed;top:0;height:1px;background:url(https://example.test/track)" onerror="alert(1)"><span style="width:50%">Text</span>`)
	require.NoError(t, err)
	images := renderedImageAttributes(t, got)
	require.Len(t, images, 1)
	assert.Equal(t, "width:50%", normalizedImageStyle(images[0]["style"]))
	assert.NotContains(t, got, "onerror")
	assert.NotContains(t, got, "position")
	assert.NotContains(t, got, "background")
	assert.NotContains(t, got, "height:")
	assert.Equal(t, 1, strings.Count(got, `style="`), "width styles must not be allowed on spans")

	for _, value := range []string{"0px", "101%", "10001px", "expression(alert(1))", "calc(50% + 1px)", "var(--width)", "url(https://example.test/track)"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			got, err := New().Render(`<img src="diagram.png" style="width:` + value + `">`)
			require.NoError(t, err)
			images := renderedImageAttributes(t, got)
			require.Len(t, images, 1)
			assert.Empty(t, images[0]["style"])
		})
	}
}

func TestImageWidthKeepsURLAndAttributeSanitization(t *testing.T) {
	t.Parallel()

	got, err := New().Render(`![Diagram](javascript:alert%281%29){width=50%}`)
	require.NoError(t, err)
	assert.NotContains(t, got, "javascript:")

	got, err = New().Render(`![Diagram](diagram.png){width=50% onerror="alert(1)"}`)
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
