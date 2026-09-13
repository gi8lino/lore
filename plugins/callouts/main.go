// Callouts is a standalone WASI plugin. It imports only the public wire types,
// never Lore's renderer, registry, persistence, or handler packages.
package main

import (
	"html"
	"strings"

	"github.com/gi8lino/lore/pluginapi"
)

// main runs the package entry point.
func main() {}

// transform converts supported callout blocks into intermediate HTML fragments.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if request.APIVersion != pluginapi.Version || request.Module != "callouts" || request.Stage != "preprocess" {
		return pluginapi.RenderResult{Error: "unsupported render request"}
	}
	lines := strings.Split(request.Source, "\n")
	var output fragments
	for index := 0; index < len(lines); index++ {
		if marker := openingFence(lines[index]); marker != "" {
			output.line(lines[index])
			for index++; index < len(lines); index++ {
				output.line(lines[index])
				if closesFence(lines[index], marker) {
					break
				}
			}
			continue
		}
		kind := calloutKind(lines[index])
		if kind == "" {
			output.line(lines[index])
			continue
		}
		var body []string
		for index++; index < len(lines) && strings.TrimSpace(lines[index]) != ""; index++ {
			body = append(body, strings.TrimSpace(lines[index]))
		}
		label := strings.ToUpper(kind[:1]) + kind[1:]
		output.line(`<aside class="callout ` + kind + `"><strong>` + html.EscapeString(label) + `</strong><div class="callout-body">`)
		markdown := strings.Join(body, "\n")
		output.flush()
		output.parts = append(output.parts, pluginapi.RenderPart{Markdown: &markdown})
		output.text("</div></aside>\n")
	}
	output.flush()
	return pluginapi.RenderResult{Parts: output.parts}
}

// calloutKind normalizes and validates a supported callout kind.
func calloutKind(line string) string {
	body, ok := strings.CutPrefix(strings.TrimSpace(line), "!!! ")
	if !ok {
		return ""
	}
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return ""
	}
	kind := strings.ToLower(fields[0])
	switch kind {
	case "note", "info", "tip", "success", "warning", "danger", "error":
		return kind
	default:
		return ""
	}
}

// fragments incrementally builds bounded render fragments for one callout transformation.
type fragments struct {
	// pending buffers literal text until the next structured fragment.
	pending strings.Builder
	// parts contains completed render fragments in output order.
	parts []pluginapi.RenderPart
	// lines tracks buffered literal line count for output limits.
	lines int
}

// line appends one literal output line.
func (f *fragments) line(value string) {
	if f.lines > 0 {
		f.text("\n")
	}
	f.text(value)
	f.lines++
}

// text appends one literal text fragment.
func (f *fragments) text(value string) { f.pending.WriteString(value) }

// flush emits buffered literal text as one fragment.
func (f *fragments) flush() {
	if f.pending.Len() == 0 {
		return
	}
	f.parts = append(f.parts, pluginapi.RenderPart{Text: f.pending.String()})
	f.pending.Reset()
}

// openingFence recognizes an opening Markdown fence and returns its marker.
func openingFence(line string) string {
	line = strings.TrimSpace(line)
	for _, delimiter := range []string{"`", "~"} {
		length := len(line) - len(strings.TrimLeft(line, delimiter))
		if length >= 3 && validFenceInfo(delimiter, line[length:]) {
			return line[:length]
		}
	}
	return ""
}

// validFenceInfo validates the optional info string on a fenced code block.
func validFenceInfo(delimiter, info string) bool {
	return delimiter != "`" || !strings.ContainsRune(info, '`')
}

// closesFence reports whether a line closes the active Markdown fence.
func closesFence(line, marker string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, marker) && strings.Trim(line, string(marker[0])) == ""
}
