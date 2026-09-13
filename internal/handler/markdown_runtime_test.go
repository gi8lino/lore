package handler

import (
	"context"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/stretchr/testify/require"
	"testing"
)

func testMarkdownRenderer(t testing.TB) *md.Renderer {
	t.Helper()
	renderer, err := md.New(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = renderer.Close(context.Background()) })
	return renderer
}
