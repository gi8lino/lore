package handler

import (
	"context"
	"testing"

	"github.com/gi8lino/lore/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type knowledgeContentStub struct {
	pages    map[string]service.Page
	snippets map[string]service.KnowledgeSnippet
}

func (s knowledgeContentStub) GetPage(_ context.Context, slug string) (service.Page, error) {
	page, ok := s.pages[slug]
	if !ok {
		return service.Page{}, service.ErrNotFound
	}
	return page, nil
}

func (s knowledgeContentStub) KnowledgeSnippetByName(
	_ context.Context,
	kind string,
	name string,
) (service.KnowledgeSnippet, error) {
	item, ok := s.snippets[kind+":"+name]
	if !ok {
		return service.KnowledgeSnippet{}, service.ErrNotFound
	}
	return item, nil
}

func TestExpandKnowledgeMarkdownReportsMissingContent(t *testing.T) {
	t.Parallel()

	_, err := expandKnowledgeMarkdown(
		context.Background(),
		knowledgeContentStub{},
		"{{snippet:missing}}",
		nil,
		0,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, service.ErrNotFound)
}

func TestExpandKnowledgeMarkdownReportsRecursiveIncludes(t *testing.T) {
	t.Parallel()

	content := knowledgeContentStub{pages: map[string]service.Page{
		"recursive": {Slug: "recursive", Markdown: "{{include:recursive}}"},
	}}
	_, err := expandKnowledgeMarkdown(
		context.Background(),
		content,
		"{{include:recursive}}",
		nil,
		0,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recursive page include")
}

func TestExpandKnowledgeMarkdownReportsDepthOverflow(t *testing.T) {
	t.Parallel()

	_, err := expandKnowledgeMarkdown(
		context.Background(),
		knowledgeContentStub{},
		"content",
		nil,
		maxKnowledgeExpansionDepth+1,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum depth")
}
