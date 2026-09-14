package pluginbrowser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPresentationStylesAreScopedColorsOnly(t *testing.T) {
	result := scopedColors("io.example.test", `
 @import url(https://evil.test/);
 @media screen { body { color: red; } }
 body, .prose .tone { background:color-mix(in srgb,var(--accent) 25%,var(--surface)); position:fixed; inset:0; display:none; }
 .bad { background:url(https://evil.test/); color:expression(evil); border-color: red; }
 .escape:has(*) { color:red; }
 .generated::before { content:'secret'; color:blue; }
 `)
	assert.Contains(t, result, `[data-lore-plugin="io.example.test"] .tone`)
	assert.Contains(t, result, "background-color:color-mix")
	assert.Contains(t, result, "border-color:red")
	for _, forbidden := range []string{"@import", "@media", "position", "inset", "display", "url(", "expression", "content:", ":has"} {
		assert.NotContains(t, result, forbidden)
	}
}
