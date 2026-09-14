package wasm

import (
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// taskListSyntax installs Goldmark task-list parsing together with Lore's inert checkbox renderer.
type taskListSyntax struct{}

// Extend installs task-list parsing and the safe presentation renderer.
func (taskListSyntax) Extend(markdown goldmark.Markdown) {
	extension.TaskList.Extend(markdown)
	markdown.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(taskCheckBoxRenderer{}, 100),
		),
	)
}

// taskCheckBoxRenderer renders task-list markers as inert, themeable controls.
// Using a span keeps raw HTML inputs outside the sanitizer allowlist while
// preserving the task state in the rendered document.
type taskCheckBoxRenderer struct{}

// RegisterFuncs implements renderer.NodeRenderer.
func (taskCheckBoxRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(extensionast.KindTaskCheckBox, renderTaskCheckBox)
}

// renderTaskCheckBox renders one task marker without making page content interactive.
func renderTaskCheckBox(
	w util.BufWriter,
	_ []byte,
	node gast.Node,
	entering bool,
) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}

	checkbox := node.(*extensionast.TaskCheckBox)
	checked := "false"
	className := "task-list-checkbox"
	mark := ""

	if checkbox.IsChecked {
		checked = "true"
		className += " checked"
		mark = "&#10003;"
	}

	_, err := w.WriteString(
		`<span class="` + className + `" role="checkbox" aria-checked="` + checked + `" aria-disabled="true">` + mark + `</span> `,
	)
	if err != nil {
		return gast.WalkStop, err
	}

	return gast.WalkContinue, nil
}
