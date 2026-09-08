package markdown

import (
	"strconv"
	"testing"
)

func TestNormalizeImageWidth(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ value, want string }{
		{"1", "1px"}, {"640", "640px"}, {"640px", "640px"},
		{"000640", "640px"}, {"10000px", "10000px"},
		{"1%", "1%"}, {"50%", "50%"}, {"100%", "100%"},
		{"", ""}, {"0", ""}, {"0px", ""}, {"0%", ""},
		{"-1", ""}, {"+1", ""}, {"10001px", ""}, {"101%", ""},
		{"50.5%", ""}, {"1.5px", ""}, {"1e2", ""}, {"0x10", ""},
		{"50%%", ""}, {"50px%", ""}, {"50%px", ""},
		{"10em", ""}, {"10vw", ""}, {"640PX", ""}, {"auto", ""},
		{" 640", ""}, {"640 ", ""}, {"640 px", ""},
		{"640\n", ""}, {"640\x00", ""}, {"\uff16\uff14\uff10", ""},
		{"999999999999999999999999999999999", ""},
		{"50%;position:fixed", ""}, {"calc(100% - 1px)", ""},
		{"url(https://example.test/image)", ""}, {"var(--width)", ""},
	} {
		t.Run(test.value, func(t *testing.T) {
			t.Parallel()
			if got := normalizeImageWidth(test.value); got != test.want {
				t.Errorf("normalizeImageWidth(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestImageWidthRanges(t *testing.T) {
	t.Parallel()

	for width := 1; width <= maxImageWidthPixels; width++ {
		value := strconv.Itoa(width)
		if got := normalizeImageWidth(value); got != value+"px" {
			t.Fatalf("pixel width %q normalized to %q", value, got)
		}
		if !validImageWidthStyle(value + "px") {
			t.Fatalf("sanitizer rejected pixel width %q", value)
		}
		if got := validImageWidthStyle(value + "%"); got != (width <= 100) {
			t.Fatalf("percentage %q validity = %t", value, got)
		}
	}
}

func TestParseImageWidthDirective(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		value, want string
		consumed    int
	}{
		{"{width=640}", "640px", len("{width=640}")},
		{"{width=640px} after", "640px", len("{width=640px}")},
		{"{width=50%}![next](b.png)", "50%", len("{width=50%}")},
		{"{width=50%}\nNext line", "50%", len("{width=50%}")},
		{"{width=50%}{width=25%}", "50%", len("{width=50%}")},
		{"", "", 0}, {"{width=", "", 0}, {"{width=50%", "", 0},
		{"{width=0}", "", 0}, {"{width=101%}", "", 0},
		{" {width=50%}", "", 0}, {"\n{width=50%}", "", 0},
		{`\{width=50%}`, "", 0}, {"&#123;width=50%}", "", 0},
		{"{width =50%}", "", 0}, {"{width=50% }", "", 0},
		{"{height=50}", "", 0}, {"{width=50% height=20}", "", 0},
		{"{width=50%;position:fixed}", "", 0},
	} {
		t.Run(test.value, func(t *testing.T) {
			t.Parallel()
			width, consumed := parseImageWidthDirective([]byte(test.value))
			if width != test.want || consumed != test.consumed {
				t.Errorf("parseImageWidthDirective(%q) = (%q, %d), want (%q, %d)",
					test.value, width, consumed, test.want, test.consumed)
			}
		})
	}
}

func TestValidImageWidthStyle(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "640", "000640px", "0px", "101%", "10001px", "50%;position:fixed", "50% !important"} {
		if validImageWidthStyle(value) {
			t.Errorf("sanitizer accepted non-canonical width %q", value)
		}
	}
}

func FuzzParseImageWidthDirective(f *testing.F) {
	for _, value := range []string{"", "{width=640}", "{width=50%} after", "{width=0}", "{width=-1}", "{width=50%;position:fixed}"} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, source string) {
		width, consumed := parseImageWidthDirective([]byte(source))
		if consumed == 0 {
			if width != "" {
				t.Fatalf("rejected directive returned width %q", width)
			}
			return
		}
		if consumed < 0 || consumed > len(source) {
			t.Fatalf("consumed %d bytes from %d-byte input", consumed, len(source))
		}
		if source[consumed-1] != '}' || !validImageWidthStyle(width) {
			t.Fatalf("invalid parsed directive: width %q, consumed %d", width, consumed)
		}
	})
}
