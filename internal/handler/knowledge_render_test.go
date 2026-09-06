package handler

import (
	"context"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type knowledgeContentStub struct {
	pages    map[string]domain.Page
	snippets map[string]domain.KnowledgeSnippet
}

func (s knowledgeContentStub) GetPage(_ context.Context, slug string) (domain.Page, error) {
	page, ok := s.pages[slug]
	if !ok {
		return domain.Page{}, domain.ErrNotFound
	}

	return page, nil
}

func (s knowledgeContentStub) KnowledgeSnippetByName(
	_ context.Context,
	kind string,
	name string,
) (domain.KnowledgeSnippet, error) {
	item, ok := s.snippets[kind+":"+name]
	if !ok {
		return domain.KnowledgeSnippet{}, domain.ErrNotFound
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
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestExpandKnowledgeMarkdownReportsRecursiveIncludes(t *testing.T) {
	t.Parallel()

	content := knowledgeContentStub{pages: map[string]domain.Page{
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

func TestExpandKnowledgeMarkdownSyntax(t *testing.T) {
	t.Parallel()
	content := knowledgeContentStub{
		snippets: map[string]domain.KnowledgeSnippet{
			"variable:site":    {Content: "Lore"},
			"snippet:greeting": {Content: "Hello"},
			"snippet:literal":  {Content: "{{var:site}}"},
			"snippet:a:b":      {Content: "colon"},
			"snippet:":         {Content: "empty name"},
		},
		pages: map[string]domain.Page{
			"guide": {Markdown: "# {{var:site}}\n{{snippet:greeting}}"},
		},
	}
	t.Run("adjacent macros", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{snippet:greeting}}{{var:site}}!", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "HelloLore!", got)
	})
	t.Run("name whitespace", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "before {{var: \t site \t}} after", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "before Lore after", got)
	})
	t.Run("colon in name", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{snippet:a:b}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "colon", got)
	})
	t.Run("empty macro stays literal", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{snippet:}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "{{snippet:}}", got)
	})
	t.Run("whitespace name retains lookup", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{snippet: }}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "empty name", got)
	})
	t.Run("unknown and incomplete", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{other:site}} {{var:site} {{var:site", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "{{other:site}} {{var:site} {{var:site", got)
	})
	t.Run("exact kind syntax", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{ var:site}} {{Var:site}} {{var :site}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "{{ var:site}} {{Var:site}} {{var :site}}", got)
	})
	t.Run("nested valid macro", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{unknown:{{var:site}}}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "{{unknown:Lore}}", got)
	})
	t.Run("overlapping opening braces", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{{var:site}}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "{Lore}", got)
	})
	t.Run("braces in name", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{var:si{te}} {{var:si}te}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "{{var:si{te}} {{var:si}te}}", got)
	})
	t.Run("valid after malformed", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{var:bad} {{var:site}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "{{var:bad} Lore", got)
	})
	t.Run("macro cannot span lines", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{var:\nsite}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "{{var:\nsite}}", got)
	})
	t.Run("snippet results are not rescanned", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{snippet:literal}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "{{var:site}}", got)
	})
	t.Run("includes expand recursively and independently", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "{{include:/guide/}}\n{{include:guide}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "# Lore\nHello\n# Lore\nHello", got)
	})
	t.Run("backtick fences", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "```md\n{{var:site}}\n```\n{{var:site}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "```md\n{{var:site}}\n```\nLore", got)
	})
	t.Run("indented tilde fences", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "  ~~~\n{{var:site}}\n  ~~~\n{{var:site}}", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "  ~~~\n{{var:site}}\n  ~~~\nLore", got)
	})
	t.Run("inline code retains existing expansion", func(t *testing.T) {
		got, err := expandKnowledgeMarkdown(context.Background(), content, "`{{var:site}}`", nil, 0)
		require.NoError(t, err)
		assert.Equal(t, "`Lore`", got)
	})
}

func TestExpandKnowledgeMarkdownRejectsEmptyInclude(t *testing.T) {
	_, err := expandKnowledgeMarkdown(context.Background(), knowledgeContentStub{}, "{{include: / }}", nil, 0)
	require.EqualError(t, err, "include requires a page path")
}
