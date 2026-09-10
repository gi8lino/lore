package handler

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	md "github.com/gi8lino/lore/internal/markdown"
)

const maxKnowledgeExpansionDepth = 5

type knowledgeContent interface {
	GetPage(context.Context, string) (domain.Page, error)
	KnowledgeSnippetByName(context.Context, string, string) (domain.KnowledgeSnippet, error)
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
func (c applicationKnowledgeContent) GetPage(ctx context.Context, slug string) (domain.Page, error) {
	return c.catalogUseCases.GetPage(ctx, slug)
}

// KnowledgeSnippetByName returns reusable content for a knowledge macro.
func (c applicationKnowledgeContent) KnowledgeSnippetByName(
	ctx context.Context,
	kind string,
	name string,
) (domain.KnowledgeSnippet, error) {
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
		return expandStoredKnowledge(ctx, content, kind, name)
	case "include":
		return expandPageInclude(ctx, content, name, seen, depth)
	default:
		return "", fmt.Errorf("unsupported knowledge macro %q", kind)
	}
}

// expandStoredKnowledge resolves a stored variable or snippet.
func expandStoredKnowledge(ctx context.Context, content knowledgeContent, kind, name string) (string, error) {
	storedKind := kind
	if kind == "var" {
		storedKind = "variable"
	}

	item, err := content.KnowledgeSnippetByName(ctx, storedKind, name)
	if errors.Is(err, domain.ErrNotFound) {
		return "", fmt.Errorf("%s %q not found: %w", kind, name, err)
	}
	if err != nil {
		return "", err
	}

	return item.Content, nil
}

// expandPageInclude resolves and recursively expands one whole-page or heading include.
func expandPageInclude(
	ctx context.Context,
	content knowledgeContent,
	name string,
	seen map[string]bool,
	depth int,
) (string, error) {
	pageTarget, heading := md.SplitHeadingTarget(name)
	slug := strings.Trim(pageTarget, "/")
	if slug == "" {
		return "", fmt.Errorf("include requires a page path")
	}

	includeKey := includeTargetKey(slug, heading)
	if seen[includeKey] {
		return "", fmt.Errorf("recursive page include %q", includeKey)
	}

	page, err := content.GetPage(ctx, slug)
	if errors.Is(err, domain.ErrNotFound) {
		return "", fmt.Errorf("included page %q not found: %w", slug, err)
	}
	if err != nil {
		return "", err
	}

	markdown, err := includedMarkdown(page.Markdown, slug, heading)
	if err != nil {
		return "", err
	}

	nextSeen := maps.Clone(seen)
	nextSeen[includeKey] = true

	expanded, err := expandKnowledgeMarkdown(ctx, content, markdown, nextSeen, depth+1)
	if err != nil {
		return "", fmt.Errorf("expand include %s: %w", slug, err)
	}

	return expanded, nil
}

// includeTargetKey returns the recursion key for a whole-page or heading include.
func includeTargetKey(slug, heading string) string {
	if heading == "" {
		return slug
	}

	return slug + "#" + md.HeadingID(heading)
}

// includedMarkdown selects the requested heading section when one is present.
func includedMarkdown(source, slug, heading string) (string, error) {
	if heading == "" {
		return source, nil
	}

	section, err := markdownSection(source, heading)
	if err != nil {
		return "", fmt.Errorf("include section %s#%s: %w", slug, heading, err)
	}

	return section, nil
}

// markdownSection returns one ATX-heading section, including its heading, through the next sibling or ancestor heading.
func markdownSection(source, requested string) (string, error) {
	requestedID := md.HeadingID(requested)
	lines := strings.Split(source, "\n")
	start := -1
	level := 0
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
		if fence != "" {
			continue
		}

		headingLevel, title, ok := atxHeading(line)
		if !ok {
			continue
		}
		if start < 0 && md.HeadingID(title) != requestedID {
			continue
		}
		if start < 0 {
			start = index
			level = headingLevel
			continue
		}
		if headingLevel <= level {
			return strings.Join(lines[start:index], "\n"), nil
		}
	}

	if start < 0 {
		return "", fmt.Errorf("heading %q not found", requested)
	}
	return strings.Join(lines[start:], "\n"), nil
}

// atxHeading parses a Markdown ATX heading without interpreting fenced code.
func atxHeading(line string) (level int, title string, ok bool) {
	trimmed := strings.TrimLeft(line, " ")
	indent := len(line) - len(trimmed)
	if indent > 3 || !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}
	for level < len(trimmed) && level < 6 && trimmed[level] == '#' {
		level++
	}
	if level == len(trimmed) || (trimmed[level] != ' ' && trimmed[level] != '\t') {
		return 0, "", false
	}
	title = strings.TrimSpace(trimmed[level:])
	for strings.HasSuffix(title, "#") {
		title = strings.TrimSpace(strings.TrimSuffix(title, "#"))
	}
	if title == "" {
		return 0, "", false
	}
	return level, title, true
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
