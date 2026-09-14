package handler

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/gi8lino/lore/internal/domain"
	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/plugin"
)

const maxKnowledgeExpansionDepth = 5

// legacyKnowledgeExpansion contains the temporary core include expansion used
// until Includes migrates to its own plugin.
type legacyKnowledgeExpansion struct {
	Markdown     string
	Replacements []plugin.Replacement
}

// expandLegacyKnowledgeMarkdown expands legacy page includes while leaving
// plugin-owned Variables and Snippets macros untouched.
func expandLegacyKnowledgeMarkdown(
	ctx context.Context,
	content knowledgeContent,
	source string,
) (legacyKnowledgeExpansion, error) {
	markdown, err := expandPageIncludes(ctx, content, source, nil, 0)
	if err != nil {
		return legacyKnowledgeExpansion{}, err
	}

	return legacyKnowledgeExpansion{Markdown: markdown}, nil
}

// knowledgeContent exposes the page lookup needed by the temporary include bridge.
type knowledgeContent interface {
	GetPage(context.Context, string) (domain.Page, error)
}

type applicationKnowledgeContent struct {
	catalogUseCases pageContentService
}

// knowledgeContentFrom adapts the page catalog to the temporary include contract.
func knowledgeContentFrom(catalog pageContentService) knowledgeContent {
	return applicationKnowledgeContent{catalogUseCases: catalog}
}

// GetPage returns page content for an include macro.
func (c applicationKnowledgeContent) GetPage(ctx context.Context, slug string) (domain.Page, error) {
	return c.catalogUseCases.GetPage(ctx, slug)
}

// expandPageIncludes expands page includes outside fenced code blocks.
func expandPageIncludes(
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
		if fence != "" || !strings.Contains(line, "{{include:") {
			continue
		}

		var output strings.Builder
		for {
			macro, ok := parseIncludeMacro(line)
			if !ok {
				output.WriteString(line)
				break
			}
			output.WriteString(line[:macro.start])
			replacement, err := expandPageInclude(ctx, content, macro.name, seen, depth)
			if err != nil {
				return "", err
			}
			output.WriteString(replacement)
			line = line[macro.end:]
		}
		lines[index] = output.String()
	}

	return strings.Join(lines, "\n"), nil
}

type includeMacro struct {
	start int
	end   int
	name  string
}

// parseIncludeMacro finds the next well-formed {{include:path}} invocation.
func parseIncludeMacro(line string) (includeMacro, bool) {
	const opening = "{{include:"
	for offset := 0; offset < len(line); {
		found := strings.Index(line[offset:], opening)
		if found < 0 {
			return includeMacro{}, false
		}
		start := offset + found
		bodyStart := start + len(opening)
		close := strings.Index(line[bodyStart:], "}}")
		if close < 0 {
			return includeMacro{}, false
		}
		end := bodyStart + close
		name := strings.TrimSpace(line[bodyStart:end])
		if name != "" && !strings.ContainsAny(name, "{}\r\n") {
			return includeMacro{start: start, end: end + 2, name: name}, true
		}
		offset = end + 2
	}
	return includeMacro{}, false
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
	expanded, err := expandPageIncludes(ctx, content, markdown, nextSeen, depth+1)
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

// markdownSection returns one ATX-heading section through the next sibling or ancestor heading.
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
