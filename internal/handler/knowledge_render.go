package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gi8lino/lore/internal/service"
)

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

		var expanded strings.Builder
		for {
			macro, ok := parseKnowledgeMacro(line)
			if !ok {
				expanded.WriteString(line)
				break
			}
			expanded.WriteString(line[:macro.start])
			replacement, err := expandKnowledgeMacro(ctx, content, macro.kind, macro.name, seen, depth)
			if err != nil {
				return "", err
			}
			expanded.WriteString(replacement)
			line = line[macro.end:]
		}
		lines[index] = expanded.String()
	}

	return strings.Join(lines, "\n"), nil
}

// knowledgeMacro identifies one supported macro within a source line.
type knowledgeMacro struct {
	start, end int
	kind, name string
}

// parseKnowledgeMacro finds the next {{kind:name}} without consuming malformed
// or unknown syntax. Names must contain at least one character and no braces.
func parseKnowledgeMacro(line string) (macro knowledgeMacro, found bool) {
	for offset := 0; offset < len(line); {
		opening := strings.Index(line[offset:], "{{")
		if opening < 0 {
			break
		}
		start := offset + opening
		offset = start + 1 // Allow a valid macro inside malformed outer braces.
		bodyStart := start + 2
		brace := strings.IndexAny(line[bodyStart:], "{}")
		if brace < 0 {
			break
		}
		end := bodyStart + brace
		if !strings.HasPrefix(line[end:], "}}") {
			continue
		}
		kind, name, ok := strings.Cut(line[bodyStart:end], ":")
		if !ok || name == "" {
			continue
		}
		switch kind {
		case "var", "snippet", "include":
			return knowledgeMacro{start: start, end: end + 2, kind: kind, name: strings.TrimSpace(name)}, true
		}
	}
	return knowledgeMacro{}, false
}

// expandKnowledgeMacro resolves one parsed macro, recursively expanding page includes.
func expandKnowledgeMacro(
	ctx context.Context,
	content knowledgeContent,
	kind, name string,
	seen map[string]bool,
	depth int,
) (string, error) {
	switch kind {
	case "var", "snippet":
		storedKind := kind
		if kind == "var" {
			storedKind = "variable"
		}
		item, err := content.KnowledgeSnippetByName(ctx, storedKind, name)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return "", fmt.Errorf("%s %q not found: %w", kind, name, err)
			}
			return "", err
		}
		return item.Content, nil
	case "include":
		slug := strings.Trim(name, "/")
		if slug == "" {
			return "", fmt.Errorf("include requires a page path")
		}
		if seen[slug] {
			return "", fmt.Errorf("recursive page include %q", slug)
		}
		page, err := content.GetPage(ctx, slug)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return "", fmt.Errorf("included page %q not found: %w", slug, err)
			}
			return "", err
		}
		nextSeen := make(map[string]bool, len(seen)+1)
		for key, value := range seen {
			nextSeen[key] = value
		}
		nextSeen[slug] = true
		expanded, err := expandKnowledgeMarkdown(ctx, content, page.Markdown, nextSeen, depth+1)
		if err != nil {
			return "", fmt.Errorf("expand include %s: %w", slug, err)
		}
		return expanded, nil
	default:
		return "", fmt.Errorf("unsupported knowledge macro %q", kind)
	}
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
