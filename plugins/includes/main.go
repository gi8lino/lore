package main

import (
	"fmt"
	"maps"
	"strings"
	"unicode"

	"github.com/gi8lino/lore/pluginapi"
	pluginmarkdown "github.com/gi8lino/lore/plugins/markdown"
)

const maxIncludeDepth = 5

// main runs the package entry point.
func main() {}

// pageLoader returns authorized page Markdown for one canonical path.
type pageLoader func(string) (pluginapi.PageContent, error)

// transform expands Includes during the content-preprocess stage.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if request.Stage != "content-preprocess" {
		return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: request.Source}}}
	}

	markdown, err := expandIncludes(request.Source, loadPage, nil, 0)
	if err != nil {
		return pluginapi.RenderResult{Error: err.Error()}
	}
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: markdown}}}
}

// loadPage resolves one page through Lore's authorized page-content capability.
func loadPage(slug string) (pluginapi.PageContent, error) {
	var result pluginapi.PageContent
	err := pluginapi.Call("pages.content", pluginapi.PageRef{Slug: slug}, &result)
	return result, err
}

// expandIncludes resolves includes outside fenced code blocks.
func expandIncludes(source string, load pageLoader, seen map[string]bool, depth int) (string, error) {
	if depth > maxIncludeDepth {
		return "", fmt.Errorf("include expansion exceeds maximum depth of %d", maxIncludeDepth)
	}
	if seen == nil {
		seen = map[string]bool{}
	}

	lines := strings.Split(source, "\n")
	fence := ""
	for index, line := range lines {
		if fence != "" {
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}
			continue
		}
		if marker := pluginmarkdown.Fence(line); marker != "" {
			fence = marker
			continue
		}
		if !strings.Contains(line, "{{include:") {
			continue
		}

		expanded, err := expandLine(line, load, seen, depth)
		if err != nil {
			return "", err
		}
		lines[index] = expanded
	}

	return strings.Join(lines, "\n"), nil
}

// expandLine replaces every well-formed include macro in one source line.
func expandLine(line string, load pageLoader, seen map[string]bool, depth int) (string, error) {
	var output strings.Builder
	for {
		macro, ok := parseInclude(line)
		if !ok {
			output.WriteString(line)
			break
		}
		output.WriteString(line[:macro.start])
		included, err := expandInclude(macro.target, load, seen, depth)
		if err != nil {
			return "", err
		}
		output.WriteString(included)
		line = line[macro.end:]
	}
	return output.String(), nil
}

type includeMacro struct {
	start  int
	end    int
	target string
}

// parseInclude finds the next valid {{include:path}} macro.
func parseInclude(line string) (includeMacro, bool) {
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
		target := strings.TrimSpace(line[bodyStart:end])
		if target != "" && !strings.ContainsAny(target, "{}\r\n") {
			return includeMacro{start: start, end: end + 2, target: target}, true
		}
		offset = end + 2
	}
	return includeMacro{}, false
}

// expandInclude resolves one whole-page or heading-section include recursively.
func expandInclude(target string, load pageLoader, seen map[string]bool, depth int) (string, error) {
	page, heading := splitHeadingTarget(target)
	slug := strings.Trim(page, "/")
	if slug == "" {
		return "", fmt.Errorf("include requires a page path")
	}
	key := includeKey(slug, heading)
	if seen[key] {
		return "", fmt.Errorf("recursive page include %q", key)
	}

	content, err := load(slug)
	if err != nil {
		return "", fmt.Errorf("include page %q: %w", slug, err)
	}
	markdown, err := includedMarkdown(content.Markdown, slug, heading)
	if err != nil {
		return "", err
	}
	nextSeen := maps.Clone(seen)
	nextSeen[key] = true
	expanded, err := expandIncludes(markdown, load, nextSeen, depth+1)
	if err != nil {
		return "", fmt.Errorf("expand include %s: %w", slug, err)
	}
	return expanded, nil
}

// splitHeadingTarget separates a page path from an optional heading fragment.
func splitHeadingTarget(target string) (string, string) {
	page, heading, _ := strings.Cut(strings.TrimSpace(target), "#")
	return strings.TrimSpace(page), strings.TrimSpace(heading)
}

// includeKey returns the recursion key for a complete page or heading section.
func includeKey(slug, heading string) string {
	if heading == "" {
		return slug
	}
	return slug + "#" + headingID(heading)
}

// includedMarkdown selects one heading section when requested.
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

// markdownSection returns an ATX heading through the next sibling or ancestor heading.
func markdownSection(source, requested string) (string, error) {
	requestedID := headingID(requested)
	lines := strings.Split(source, "\n")
	start := -1
	level := 0
	fence := ""
	for index, line := range lines {
		if fence != "" {
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}
			continue
		}
		if marker := pluginmarkdown.Fence(line); marker != "" {
			fence = marker
			continue
		}
		headingLevel, title, ok := atxHeading(line)
		if !ok {
			continue
		}
		if start < 0 && headingID(title) != requestedID {
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

// atxHeading parses one Markdown ATX heading.
func atxHeading(line string) (level int, title string, ok bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || !strings.HasPrefix(trimmed, "#") {
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
	return level, title, title != ""
}

// headingID mirrors Lore's stable ASCII heading-anchor normalization.
func headingID(value string) string {
	value = strings.ReplaceAll(value, "/", " ")
	value = strings.TrimSpace(value)
	var output strings.Builder
	separator := false
	for _, character := range value {
		character = unicode.ToLower(character)
		if isHeadingRune(character) {
			if separator && output.Len() > 0 {
				output.WriteByte('-')
			}
			separator = false
			output.WriteRune(character)
		} else {
			separator = true
		}
	}
	return strings.Trim(output.String(), "-")
}

// isHeadingRune reports whether Lore preserves a character in heading anchors.
func isHeadingRune(character rune) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		character == '_' || character == '-'
}
