// Package callouts implements Lore callout blocks through the module API.
package callouts

import (
	stdhtml "html"
	"strings"

	"github.com/gi8lino/lore/internal/markdown/blocksyntax"
	"github.com/gi8lino/lore/internal/plugin"
)

const ID = "io.lore.callouts"

// Module transforms callout blocks using only the render capability.
type Module struct{}

func Descriptor() plugin.Descriptor {
	return plugin.Descriptor{ID: ID, Name: "Callouts", Description: "Note, warning, and other callout blocks", DefaultEnabled: true}
}

// Preprocess converts supported callout syntax into intermediate HTML blocks.
// Lore core sanitizes this output after the full rendering pipeline.
func (Module) Preprocess(
	ctx plugin.Context,
	source string,
) (string, error) {
	if !ctx.Features[ID] {
		return source, nil
	}
	lines := strings.Split(source, "\n")
	out := make([]string, 0, len(lines))

	for index := 0; index < len(lines); index++ {
		if marker := blocksyntax.Fence(lines[index]); marker != "" {
			next := blocksyntax.AppendFence(
				lines,
				index,
				marker,
				&out,
			)

			index = next - 1
			continue
		}

		line := strings.TrimSpace(lines[index])
		kind := ""

		if after, ok := strings.CutPrefix(line, "!!! "); ok {
			parts := strings.Fields(after)

			if len(parts) > 0 {
				kind = strings.ToLower(parts[0])
			}
		}

		if !supportedCallout(kind) {
			out = append(out, lines[index])
			continue
		}

		body := make([]string, 0)

		index++

		for index < len(lines) &&
			strings.TrimSpace(lines[index]) != "" {
			body = append(
				body,
				strings.TrimSpace(lines[index]),
			)
			index++
		}

		bodyHTML, err := ctx.RenderMarkdown(strings.Join(body, "\n"))
		if err != nil {
			return "", err
		}

		label :=
			strings.ToUpper(kind[:1]) +
				kind[1:]

		out = append(
			out,
			`<aside class="callout `+
				kind+
				`"><strong>`+
				stdhtml.EscapeString(label)+
				`</strong><div class="callout-body">`+
				bodyHTML+
				`</div></aside>`+
				"\n",
		)
	}

	return strings.Join(out, "\n"), nil
}

// supportedCallout reports whether a callout kind has built-in presentation styling.
func supportedCallout(kind string) bool {
	switch kind {
	case "note",
		"info",
		"tip",
		"success",
		"warning",
		"danger",
		"error":
		return true
	default:
		return false
	}
}
