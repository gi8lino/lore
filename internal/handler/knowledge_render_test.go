package handler

import (
	"context"
	"testing"

	"github.com/gi8lino/lore/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type knowledgeContentStub struct {
	pages    map[string]model.Page
	snippets map[string]model.KnowledgeSnippet
}

func (s knowledgeContentStub) GetPage(_ context.Context, slug string) (model.Page, error) {
	page, ok := s.pages[slug]
	if !ok {
		return model.Page{}, model.ErrNotFound
	}

	return page, nil
}

func (s knowledgeContentStub) KnowledgeSnippetByName(
	_ context.Context,
	kind string,
	name string,
) (model.KnowledgeSnippet, error) {
	item, ok := s.snippets[kind+":"+name]
	if !ok {
		return model.KnowledgeSnippet{}, model.ErrNotFound
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
	assert.ErrorIs(t, err, model.ErrNotFound)
}

func TestExpandKnowledgeMarkdownReportsRecursiveIncludes(t *testing.T) {
	t.Parallel()

	content := knowledgeContentStub{pages: map[string]model.Page{
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
		snippets: map[string]model.KnowledgeSnippet{
			"variable:site":    {Content: "Lore"},
			"snippet:greeting": {Content: "Hello"},
			"snippet:literal":  {Content: "{{var:site}}"},
			"snippet:a:b":      {Content: "colon"},
			"snippet:":         {Content: "empty name"},
		},
		pages: map[string]model.Page{
			"guide": {Markdown: "# {{var:site}}\n{{snippet:greeting}}"},
		},
	}
	for _, tt := range []struct{ name, source, want string }{
		{"adjacent macros", "{{snippet:greeting}}{{var:site}}!", "HelloLore!"},
		{"name whitespace", "before {{var: \t site \t}} after", "before Lore after"},
		{"colon in name", "{{snippet:a:b}}", "colon"},
		{"empty macro stays literal", "{{snippet:}}", "{{snippet:}}"},
		{"whitespace name retains lookup", "{{snippet: }}", "empty name"},
		{"unknown and incomplete", "{{other:site}} {{var:site} {{var:site", "{{other:site}} {{var:site} {{var:site"},
		{"exact kind syntax", "{{ var:site}} {{Var:site}} {{var :site}}", "{{ var:site}} {{Var:site}} {{var :site}}"},
		{"nested valid macro", "{{unknown:{{var:site}}}}", "{{unknown:Lore}}"},
		{"overlapping opening braces", "{{{var:site}}}", "{Lore}"},
		{"braces in name", "{{var:si{te}} {{var:si}te}}", "{{var:si{te}} {{var:si}te}}"},
		{"valid after malformed", "{{var:bad} {{var:site}}", "{{var:bad} Lore"},
		{"macro cannot span lines", "{{var:\nsite}}", "{{var:\nsite}}"},
		{"snippet results are not rescanned", "{{snippet:literal}}", "{{var:site}}"},
		{"includes expand recursively and independently", "{{include:/guide/}}\n{{include:guide}}", "# Lore\nHello\n# Lore\nHello"},
		{"backtick fences", "```md\n{{var:site}}\n```\n{{var:site}}", "```md\n{{var:site}}\n```\nLore"},
		{"indented tilde fences", "  ~~~\n{{var:site}}\n  ~~~\n{{var:site}}", "  ~~~\n{{var:site}}\n  ~~~\nLore"},
		{"inline code retains existing expansion", "`{{var:site}}`", "`Lore`"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandKnowledgeMarkdown(context.Background(), content, tt.source, nil, 0)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExpandKnowledgeMarkdownRejectsEmptyInclude(t *testing.T) {
	_, err := expandKnowledgeMarkdown(context.Background(), knowledgeContentStub{}, "{{include: / }}", nil, 0)
	require.EqualError(t, err, "include requires a page path")
}
