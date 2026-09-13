package markdown

import (
	"html"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
