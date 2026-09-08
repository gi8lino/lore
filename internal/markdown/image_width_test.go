package markdown

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
