package wasm

import (
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// syntaxModule selects a public standard grammar through the same manifest for
// every distribution source. Feature flags control each fresh parser instance.
type syntaxModule struct{ owner, id, syntax string }

func (m syntaxModule) Extension(ctx plugin.Context) goldmark.Extender {
	if enabled, ok := ctx.Features[m.owner]; ok && !enabled {
		return noSyntax{}
	}
	if enabled, ok := ctx.Features[m.owner+"."+m.id]; ok && !enabled {
		return noSyntax{}
	}
	if m.syntax == "task-list" {
		return taskListSyntax{}
	}

	return map[string]goldmark.Extender{
		"tables": extension.Table, "strikethrough": extension.Strikethrough,
		"definition-list": extension.DefinitionList, "footnote": extension.Footnote,
		"linkify": extension.Linkify,
	}[m.syntax]
}

type noSyntax struct{}

func (noSyntax) Extend(goldmark.Markdown) {}
