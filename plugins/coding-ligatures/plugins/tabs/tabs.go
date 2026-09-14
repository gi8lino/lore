package main

import (
	stdhtml "html"
	"strconv"
	"strings"

	"github.com/gi8lino/lore/pluginapi"
	pluginmarkdown "github.com/gi8lino/lore/plugins/markdown"
)

// tabSection contains one parsed tab label and its de-indented Markdown body.
type tabSection struct {
	title string
	body  string
}

// transformTabs replaces top-level tab groups while keeping each panel body as
// host-rendered Markdown. This preserves nested Lore syntax without giving the
// plugin direct access to the renderer.
func transformTabs(source string) []pluginapi.RenderPart {
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

		title, ok := parseTabTitle(lines[index])
		if !ok {
			plain = append(plain, lines[index])
			index++
			continue
		}

		sections := make([]tabSection, 0, 2)
		for ok {
			bodyLines, next := indentedBody(lines, index+1)
			sections = append(sections, tabSection{title: title, body: strings.Join(bodyLines, "\n")})
			index = next
			if index >= len(lines) {
				break
			}
			title, ok = parseTabTitle(lines[index])
		}

		flushPlain()
		appendText(&parts, "\n<div class=\"markdown-tabs\"><div class=\"markdown-tab-list\" role=\"tablist\">")
		for sectionIndex, section := range sections {
			class := "markdown-tab"
			selected := "false"
			if sectionIndex == 0 {
				class += " active"
				selected = "true"
			}
			appendText(&parts, `<button type="button" class="`+class+`" role="tab" aria-selected="`+selected+`">`+stdhtml.EscapeString(section.title)+`</button>`)
		}
		appendText(&parts, `</div><div class="markdown-tab-panels">`)

		for sectionIndex, section := range sections {
			class := "markdown-tab-panel"
			if sectionIndex != 0 {
				class += " markdown-tab-panel-hidden"
			}
			appendText(&parts, `<div class="`+class+`" role="tabpanel">`)
			body := section.body
			parts = append(parts, pluginapi.RenderPart{Markdown: &body})
			appendText(&parts, `</div>`)
		}
		appendText(&parts, "</div></div>\n")
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

// parseTabTitle parses a top-level declaration such as === "Linux".
func parseTabTitle(line string) (string, bool) {
	if strings.TrimLeft(line, " \t") != line {
		return "", false
	}

	remaining, ok := strings.CutPrefix(strings.TrimSpace(line), "===")
	if !ok {
		return "", false
	}

	return parseQuotedTitle(strings.TrimSpace(remaining))
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
