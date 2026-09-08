package markdown

import (
	"html"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageTextAttributesSurviveSanitization(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, alt, title string
	}{
		{"ampersands", "A & B", "Overview & details"},
		{"literal width syntax", "Outer Inner{width=50%}", "Example {width=640}"},
		{"punctuation", "A: B; C? D=E | F + G @ H", "50% / $20 #1 ~ result"},
		{"quotes and angle brackets", `"Quoted" <diagram> & 'label'`, `<title> "quoted" & 'single'`},
		{"unicode", "Gr\u00f6sse \u2192 50% \U0001f4f7", "\u6982\u8981 & \u8a73\u7d30"},
		{"empty text", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// Isolate the sanitizer from Goldmark to catch restrictive attribute
			// policies even when the Markdown parser already emits correct HTML.
			source := `<img src="diagram.png" alt="` + html.EscapeString(test.alt) +
				`" title="` + html.EscapeString(test.title) + `">`
			got := New().sanitizer.Sanitize(source)
			images := renderedImageAttributes(t, got)

			require.Len(t, images, 1)
			assert.Equal(t, map[string]string{
				"src": "diagram.png", "alt": test.alt, "title": test.title,
			}, images[0])
		})
	}
}

func TestImageTextAttributesWithAndWithoutWidths(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, source, alt, width string
	}{
		{
			"unsized image",
			`![A & B](diagram.png "Overview & details")`,
			"A & B", "",
		},
		{
			"sized image",
			`![A & B](diagram.png "Overview & details"){width=640}`,
			"A & B", "width:640px",
		},
		{
			"width-like alt text",
			`![Diagram {width=50%}](diagram.png "Overview & details"){width=640}`,
			"Diagram {width=50%}", "width:640px",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := New().Render(test.source)
			require.NoError(t, err)
			images := renderedImageAttributes(t, got)

			require.Len(t, images, 1)
			assert.Equal(t, "diagram.png", images[0]["src"])
			assert.Equal(t, test.alt, images[0]["alt"])
			assert.Equal(t, "Overview & details", images[0]["title"])
			assert.Equal(t, test.width, normalizedImageStyle(images[0]["style"]))
		})
	}
}

func TestImageTextAttributesCannotInjectMarkup(t *testing.T) {
	t.Parallel()

	const alt = `"><script>alert(1)</script><img src=x onerror=alert(2)>`
	const title = `" onmouseover="alert(3)" & {width=50%}`
	source := `<img src="diagram.png" alt="` + html.EscapeString(alt) +
		`" title="` + html.EscapeString(title) +
		`" onerror="alert(4)" onload="alert(5)" style="width:50%;position:fixed">` +
		`<script>alert(6)</script>`

	got, err := New().RenderResolvedWithOptions(source, Slug, Options{})
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
