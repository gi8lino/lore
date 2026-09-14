package main

import (
	stdhtml "html"
	"strconv"
	"strings"

	"github.com/gi8lino/lore/pluginapi"
	pluginmarkdown "github.com/gi8lino/lore/plugins/markdown"
)

// transformDetails replaces top-level ??? blocks while keeping each body as
// host-rendered Markdown so nested Lore syntax follows the normal pipeline.
func transformDetails(source string) []pluginapi.RenderPart {
	lines := strings.Split(source, "\n")
	plain := make([]string, 0, len(lines))
	parts := make([]pluginapi.RenderPart, 0)

	flushPlain := func() {
		if len(plain) == 0 {
			return
		}
		appendText(&parts, strings.Join(plain, "\n"))
		plain = plain[:0]
	}

	for index := 0; index < len(lines); {
		if marker := pluginmarkdown.Fence(lines[index]); marker != "" {
			index = pluginmarkdown.AppendFence(lines, index, marker, &plain)
			continue
		}

		title, open, ok := parseDetailsTitle(lines[index])
		if !ok {
			plain = append(plain, lines[index])
			index++
			continue
		}

		bodyLines, next := indentedBody(lines, index+1)
		flushPlain()

		openAttribute := ""
		if open {
			openAttribute = " open"
		}
		appendText(&parts, "\n<details class=\"markdown-details\""+openAttribute+`><summary>`+stdhtml.EscapeString(title)+`</summary><div class="markdown-details-body">`)
		body := strings.Join(bodyLines, "\n")
		parts = append(parts, pluginapi.RenderPart{Markdown: &body})
		appendText(&parts, "</div></details>\n")
		index = next
	}

	flushPlain()
	if len(parts) == 0 {
		return []pluginapi.RenderPart{{Text: source}}
	}

	return parts
}

// appendText appends literal output and coalesces adjacent text fragments.
func appendText(parts *[]pluginapi.RenderPart, text string) {
	if text == "" {
		return
	}
	if len(*parts) != 0 && (*parts)[len(*parts)-1].Markdown == nil {
		(*parts)[len(*parts)-1].Text += text
		return
	}
	*parts = append(*parts, pluginapi.RenderPart{Text: text})
}

// parseDetailsTitle parses ??? and ???+ declarations.
func parseDetailsTitle(line string) (title string, open bool, ok bool) {
	if strings.TrimLeft(line, " \t") != line {
		return "", false, false
	}

	trimmed := strings.TrimSpace(line)
	remaining, open := strings.CutPrefix(trimmed, "???+")
	if !open {
		var found bool
		remaining, found = strings.CutPrefix(trimmed, "???")
		if !found {
			return "", false, false
		}
	}

	title, ok = parseQuotedTitle(strings.TrimSpace(remaining))
	return title, open, ok
}

// parseQuotedTitle parses one non-empty Go-style quoted title.
func parseQuotedTitle(value string) (string, bool) {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", false
	}

	title, err := strconv.Unquote(value)
	if err != nil || strings.TrimSpace(title) == "" {
		return "", false
	}

	return title, true
}

// indentedBody collects blank and four-space- or tab-indented body lines.
func indentedBody(lines []string, start int) ([]string, int) {
	body := make([]string, 0)
	index := start

	for index < len(lines) {
		if strings.TrimSpace(lines[index]) == "" {
			body = append(body, "")
			index++
			continue
		}

		line, ok := stripBlockIndent(lines[index])
		if !ok {
			break
		}
		body = append(body, line)
		index++
	}

	return body, index
}

// stripBlockIndent removes one supported custom-block indentation level.
func stripBlockIndent(line string) (string, bool) {
	if content, ok := strings.CutPrefix(line, "\t"); ok {
		return content, true
	}
	if content, ok := strings.CutPrefix(line, "    "); ok {
		return content, true
	}
	return "", false
}
