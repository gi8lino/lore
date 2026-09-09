package handler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gi8lino/lore/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func variableTestContent() knowledgeContentStub {
	return knowledgeContentStub{
		pages: map[string]domain.Page{"child": {Slug: "child", Markdown: "Inside {{var:environment}} and {{var:contact}}."}},
		snippets: map[string]domain.KnowledgeSnippet{
			"variable:environment": {Kind: "variable", Name: "environment", Content: "production", Description: "Deployment target"},
			"variable:contact":     {Kind: "variable", Name: "contact", Content: "support@example.test"},
			"snippet:literal":      {Kind: "snippet", Name: "literal", Content: "{{var:environment}}"},
		},
	}
}

func TestPageVariables(t *testing.T) {
	t.Parallel()
	t.Run("counts distinct variables and occurrences through includes", func(t *testing.T) {
		t.Parallel()
		expanded, err := expandPageKnowledge(context.Background(), variableTestContent(), "{{var:environment}} {{var:environment}} {{include:child}}", nil, false)
		require.NoError(t, err)
		require.Len(t, expanded.Variables, 2)
		assert.Equal(t, pageVariable{Name: "environment", Value: "production", Description: "Deployment target", Occurrences: 3}, expanded.Variables[0])
		assert.Equal(t, "contact", expanded.Variables[1].Name)
		assert.Equal(t, 1, expanded.Variables[1].Occurrences)
		assert.Equal(t, "production production Inside production and support@example.test.", expanded.Markdown)
	})
	t.Run("groups case variants using the stored variable name", func(t *testing.T) {
		t.Parallel()
		content := variableTestContent()
		content.snippets["variable:ENVIRONMENT"] = content.snippets["variable:environment"]
		expanded, err := expandPageKnowledge(context.Background(), content,
			"{{var:ENVIRONMENT}} {{var:environment}} {{var:ENVIRONMENT}}",
			map[string]string{"environment": "staging"}, false)
		require.NoError(t, err)
		require.Len(t, expanded.Variables, 1)
		assert.Equal(t, "environment", expanded.Variables[0].Name)
		assert.Equal(t, 3, expanded.Variables[0].Occurrences)
		assert.Equal(t, "staging staging staging", expanded.Markdown)
	})
	t.Run("keeps fenced examples literal and expands inline code", func(t *testing.T) {
		t.Parallel()
		source := "```text\n{{var:environment}}\n```\n`{{var:environment}}`"
		expanded, err := expandPageKnowledge(context.Background(), variableTestContent(), source, nil, false)
		require.NoError(t, err)
		require.Len(t, expanded.Variables, 1)
		assert.Equal(t, 1, expanded.Variables[0].Occurrences)
		assert.Equal(t, "```text\n{{var:environment}}\n```\n`production`", expanded.Markdown)
	})
	t.Run("retains existing nonrecursive snippet behavior", func(t *testing.T) {
		t.Parallel()
		expanded, err := expandPageKnowledge(context.Background(), variableTestContent(), "{{snippet:literal}}", nil, false)
		require.NoError(t, err)
		assert.Equal(t, "{{var:environment}}", expanded.Markdown)
		assert.Empty(t, expanded.Variables)
	})
	t.Run("does not change ordinary matching text or saved values", func(t *testing.T) {
		t.Parallel()
		content := variableTestContent()
		overrides := map[string]string{"environment": "staging"}
		expanded, err := expandPageKnowledge(context.Background(), content, "production {{var:environment}} {{include:child}}", overrides, false)
		require.NoError(t, err)
		assert.Equal(t, "production staging Inside staging and support@example.test.", expanded.Markdown)
		assert.Equal(t, "production", content.snippets["variable:environment"].Content)
		assert.Equal(t, "production", expanded.Variables[0].Value)
		assert.Equal(t, map[string]string{"environment": "staging"}, overrides)
		fresh, err := expandPageKnowledge(context.Background(), content, "{{var:environment}}", nil, false)
		require.NoError(t, err)
		assert.Equal(t, "production", fresh.Markdown)
	})
	t.Run("accepts an explicitly empty replacement", func(t *testing.T) {
		t.Parallel()
		expanded, err := expandPageKnowledge(context.Background(), variableTestContent(), "[{{var:environment}}]", map[string]string{"environment": ""}, false)
		require.NoError(t, err)
		assert.Equal(t, "[]", expanded.Markdown)
	})
	t.Run("preserves whitespace in replacements", func(t *testing.T) {
		t.Parallel()
		expanded, err := expandPageKnowledge(context.Background(), variableTestContent(), "[{{var:environment}}]", map[string]string{"environment": " staging \n"}, false)
		require.NoError(t, err)
		assert.Equal(t, "[ staging \n]", expanded.Markdown)
	})
	t.Run("does not evaluate macro syntax supplied as an override", func(t *testing.T) {
		t.Parallel()
		expanded, err := expandPageKnowledge(context.Background(), variableTestContent(), "{{var:environment}}", map[string]string{"environment": "{{include:private}} {{var:contact}}"}, false)
		require.NoError(t, err)
		assert.Equal(t, "{{include:private}} {{var:contact}}", expanded.Markdown)
	})
	t.Run("rejects an unused variable even when it exists", func(t *testing.T) {
		t.Parallel()
		_, err := expandPageKnowledge(context.Background(), variableTestContent(), "{{var:environment}}", map[string]string{"contact": "other"}, false)
		require.Error(t, err)
		var validation *domain.ValidationError
		require.ErrorAs(t, err, &validation)
		require.NotEmpty(t, validation.Fields)
		assert.Equal(t, "variables", validation.Fields[0].Field)
	})
	t.Run("does not list fenced-only variables", func(t *testing.T) {
		t.Parallel()
		_, err := expandPageKnowledge(context.Background(), variableTestContent(), "~~~\n{{var:environment}}\n~~~", map[string]string{"environment": "staging"}, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Only variables used")
	})
	t.Run("creates provenance tokens only for actual macros", func(t *testing.T) {
		t.Parallel()
		expanded, err := expandPageKnowledge(context.Background(), variableTestContent(), "production {{var:environment}}", nil, true)
		require.NoError(t, err)
		require.Len(t, expanded.Annotations, 1)
		assert.Equal(t, "production "+expanded.Annotations[0].Token, expanded.Markdown)
		assert.Equal(t, "environment", expanded.Annotations[0].Name)
		assert.Equal(t, "production", expanded.Annotations[0].Value)
	})
	t.Run("plain pages have no variable controls", func(t *testing.T) {
		t.Parallel()
		expanded, err := expandPageKnowledge(context.Background(), knowledgeContentStub{}, "A normal page", nil, true)
		require.NoError(t, err)
		assert.Empty(t, expanded.Variables)
		assert.Empty(t, expanded.Annotations)
		assert.Equal(t, "A normal page", expanded.Markdown)
	})
}

func TestVariableOverrideLimits(t *testing.T) {
	t.Parallel()
	t.Run("accepts the per-value byte limit", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, validateVariableOverrides(map[string]string{"name": strings.Repeat("x", maxExportVariableBytes)}))
	})
	t.Run("rejects an oversized UTF-8 value by bytes", func(t *testing.T) {
		t.Parallel()
		assert.ErrorContains(t, validateVariableOverrides(map[string]string{"name": strings.Repeat("\u00e4", maxExportVariableBytes/2+1)}), "8 KiB")
	})
	t.Run("rejects too many variables", func(t *testing.T) {
		t.Parallel()
		values := map[string]string{}
		for index := range maxExportVariables + 1 {
			values[fmt.Sprint(index)] = "x"
		}
		assert.ErrorContains(t, validateVariableOverrides(values), "128 variables")
	})
	t.Run("rejects an oversized total", func(t *testing.T) {
		t.Parallel()
		values := map[string]string{}
		for index := range 9 {
			values[fmt.Sprint(index)] = strings.Repeat("x", maxExportVariableBytes)
		}
		assert.ErrorContains(t, validateVariableOverrides(values), "64 KiB")
	})
	t.Run("rejects null characters", func(t *testing.T) {
		t.Parallel()
		assert.ErrorContains(t, validateVariableOverrides(map[string]string{"name": "hello\x00world"}), "null characters")
	})
	t.Run("rejects invalid UTF-8", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, validateVariableOverrides(map[string]string{"name": string([]byte{0xff})}))
	})
	t.Run("rejects noncanonical names", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, validateVariableOverrides(map[string]string{" name ": "value"}))
	})
}
