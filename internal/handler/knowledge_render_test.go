package handler

import (
	"context"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type knowledgeContentStub struct {
	pages map[string]domain.Page
}

func (s knowledgeContentStub) GetPage(_ context.Context, slug string) (domain.Page, error) {
	page, ok := s.pages[slug]
	if !ok {
		return domain.Page{}, domain.ErrNotFound
	}

	return page, nil
}

func TestExpandPageIncludesHeadingSection(t *testing.T) {
	t.Parallel()

	content := knowledgeContentStub{pages: map[string]domain.Page{
		"runbook": {Slug: "runbook", Markdown: "# Runbook\n\nIntro\n\n## Restore\n\nStep one.\n\n### Details\n\nMore.\n\n## Verify\n\nDone.\n"},
	}}

	expanded, err := expandPageIncludes(context.Background(), content, "{{include:runbook#Restore}}", nil, 0)

	require.NoError(t, err)
	assert.Contains(t, expanded, "## Restore")
	assert.Contains(t, expanded, "### Details")
	assert.NotContains(t, expanded, "## Verify")
}

func TestExpandPageIncludesLeavesPluginMacrosLiteral(t *testing.T) {
	t.Parallel()

	content := knowledgeContentStub{pages: map[string]domain.Page{
		"guide": {Slug: "guide", Markdown: "{{var:site}}\n{{snippet:greeting}}"},
	}}

	expanded, err := expandPageIncludes(context.Background(), content, "{{include:guide}}", nil, 0)

	require.NoError(t, err)
	assert.Equal(t, "{{var:site}}\n{{snippet:greeting}}", expanded)
}

func TestExpandPageIncludesRejectsRecursiveIncludes(t *testing.T) {
	t.Parallel()

	content := knowledgeContentStub{pages: map[string]domain.Page{
		"recursive": {Slug: "recursive", Markdown: "{{include:recursive}}"},
	}}

	_, err := expandPageIncludes(context.Background(), content, "{{include:recursive}}", nil, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recursive page include")
}

func TestExpandPageIncludesIgnoresFencedCode(t *testing.T) {
	t.Parallel()

	expanded, err := expandPageIncludes(
		context.Background(),
		knowledgeContentStub{},
		"```\n{{include:missing}}\n```",
		nil,
		0,
	)

	require.NoError(t, err)
	assert.Equal(t, "```\n{{include:missing}}\n```", expanded)
}
