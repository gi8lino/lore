package handler

import (
	"testing"

	md "github.com/gi8lino/lore/internal/markdown"
)

// testMarkdownRenderer returns a core renderer without starting the plugin runtime.
func testMarkdownRenderer(t testing.TB) *md.Renderer {
	t.Helper()
	return md.NewWithRegistry(nil)
}
