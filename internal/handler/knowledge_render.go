package handler

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gi8lino/lore/internal/service"
)

var knowledgeMacroPattern = regexp.MustCompile(`\{\{(var|snippet|include):([^{}]+)\}\}`)

const maxKnowledgeExpansionDepth = 5

type knowledgeContent interface {
	GetPage(context.Context, string) (service.Page, error)
	KnowledgeSnippetByName(context.Context, string, string) (service.KnowledgeSnippet, error)
}

type applicationKnowledgeContent struct {
	catalogUseCases   pageContentService
	knowledgeUseCases knowledgeContentService
}

// knowledgeContentFrom adapts focused application services to the macro expansion contract.
func knowledgeContentFrom(catalog pageContentService, knowledge knowledgeContentService) knowledgeContent {
	return applicationKnowledgeContent{catalogUseCases: catalog, knowledgeUseCases: knowledge}
}

// GetPage returns page content for an include macro.
func (c applicationKnowledgeContent) GetPage(ctx context.Context, slug string) (service.Page, error) {
	return c.catalogUseCases.GetPage(ctx, slug)
}

// KnowledgeSnippetByName returns reusable content for a knowledge macro.
func (c applicationKnowledgeContent) KnowledgeSnippetByName(
	ctx context.Context,
	kind string,
	name string,
) (service.KnowledgeSnippet, error) {
	return c.knowledgeUseCases.KnowledgeSnippetByName(ctx, kind, name)
}

// expandKnowledgeMarkdown expands trusted reusable wiki macros outside fenced code blocks.
func expandKnowledgeMarkdown(
	ctx context.Context,
	content knowledgeContent,
	source string,
	seen map[string]bool,
	depth int,
) (string, error) {
	if depth > maxKnowledgeExpansionDepth {
		return "", fmt.Errorf("knowledge expansion exceeds maximum depth of %d", maxKnowledgeExpansionDepth)
	}
	if seen == nil {
		seen = map[string]bool{}
	}
	lines := strings.Split(source, "\n")
	fence := ""
	for index, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if marker := renderFenceMarker(trimmed); marker != "" {
			if fence == "" {
				fence = marker
			} else if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if fence != "" || !strings.Contains(line, "{{") {
			continue
		}
		var expandErr error
		lines[index] = knowledgeMacroPattern.ReplaceAllStringFunc(line, func(raw string) string {
			if expandErr != nil {
				return raw
			}
			match := knowledgeMacroPattern.FindStringSubmatch(raw)
			if len(match) != 3 {
				return raw
			}
			kind := match[1]
			name := strings.TrimSpace(match[2])
			switch kind {
			case "var", "snippet":
				storedKind := kind
				if kind == "var" {
					storedKind = "variable"
				}
				item, err := content.KnowledgeSnippetByName(ctx, storedKind, name)
				if err != nil {
					if errors.Is(err, service.ErrNotFound) {
						expandErr = fmt.Errorf("%s %q not found: %w", kind, name, err)
						return raw
					}
					expandErr = err
					return raw
				}
				return item.Content
			case "include":
				slug := strings.Trim(name, "/")
				if slug == "" {
					expandErr = fmt.Errorf("include requires a page path")
					return raw
				}
				if seen[slug] {
					expandErr = fmt.Errorf("recursive page include %q", slug)
					return raw
				}
				page, err := content.GetPage(ctx, slug)
				if err != nil {
					if errors.Is(err, service.ErrNotFound) {
						expandErr = fmt.Errorf("included page %q not found: %w", slug, err)
						return raw
					}
					expandErr = err
					return raw
				}
				nextSeen := make(map[string]bool, len(seen)+1)
				for key, value := range seen {
					nextSeen[key] = value
				}
				nextSeen[slug] = true
				expanded, err := expandKnowledgeMarkdown(ctx, content, page.Markdown, nextSeen, depth+1)
				if err != nil {
					expandErr = fmt.Errorf("expand include %s: %w", slug, err)
					return raw
				}
				return expanded
			}
			return raw
		})
		if expandErr != nil {
			return "", expandErr
		}
	}
	return strings.Join(lines, "\n"), nil
}

// renderFenceMarker returns the Markdown fence opened by a line, if any.
func renderFenceMarker(line string) string {
	if strings.HasPrefix(line, "```") {
		return "```"
	}
	if strings.HasPrefix(line, "~~~") {
		return "~~~"
	}
	return ""
}
