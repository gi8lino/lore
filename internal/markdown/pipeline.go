package markdown

import (
	"context"
	"crypto/rand"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugincap"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// renderPipeline pins an active contribution set for the whole document,
// including nested Markdown and the variable-provenance rendering pass.
type renderPipeline struct {
	// context carries cancellation through the complete render.
	context context.Context
	// capabilities contains request-local host capabilities exposed to plugins.
	capabilities map[string]plugin.Capability
	// snapshot pins the active plugin contribution set for this render.
	snapshot plugin.Snapshot
	// features contains presentation feature flags visible to plugin modules.
	features map[string]bool
	// macros contains request-local macro render bindings.
	macros map[string]plugin.MacroRenderer
}

// macroInvocation records one deferred macro expansion and its owner.
type macroInvocation struct {
	// owner identifies the plugin that recognized this invocation.
	owner string
	// macro is the contribution that will render the deferred invocation.
	macro plugin.Macro
	// arguments contains serialized macro arguments.
	arguments plugin.Invocation
	// placeholder marks the invocation position in intermediate HTML.
	placeholder string
}

// newRenderPipeline binds one leased plugin snapshot to request-local rendering state.
func newRenderPipeline(snapshot plugin.Snapshot, features map[string]bool, functions Functions) *renderPipeline {
	capabilities := plugincap.Capabilities(nil, nil)
	maps.Copy(capabilities, functions.Capabilities)

	return &renderPipeline{context: functions.Context, capabilities: capabilities, snapshot: snapshot, features: maps.Clone(features), macros: maps.Clone(functions.Macros)}
}

// moduleContext builds the request-local context passed to plugin contributions.
func (r *Renderer) moduleContext(resolve func(string) string, options Options) plugin.Context {
	return plugin.Context{
		Context:        options.pipeline.context,
		Capabilities:   maps.Clone(options.pipeline.capabilities),
		Features:       maps.Clone(options.pipeline.features),
		Macros:         maps.Clone(options.pipeline.macros),
		RenderMarkdown: func(source string) (string, error) { return r.renderRawResolved(source, resolve, options) },
	}
}

// preprocess runs active plugin preprocessors in registry order.
func (p *renderPipeline) preprocess(source string, ctx plugin.Context) (string, error) {
	for _, entry := range p.snapshot.Entries {
		for _, module := range entry.Contributions.Preprocessors {
			var err error
			source, err = plugin.Guard(entry.Descriptor.ID, func() (string, error) { return module.Preprocess(ctx, source) })
			if err != nil {
				return "", err
			}
		}
	}

	return source, nil
}

// extensions creates active Goldmark extensions for the current render.
func (p *renderPipeline) extensions(ctx plugin.Context) ([]goldmark.Extender, error) {
	var result []goldmark.Extender
	for _, entry := range p.snapshot.Entries {
		for _, module := range entry.Contributions.MarkdownExtensions {
			extender, err := plugin.Guard(entry.Descriptor.ID, func() (goldmark.Extender, error) { return module.Extension(ctx), nil })
			if err != nil {
				return nil, err
			}
			if extender == nil {
				return nil, fmt.Errorf("plugin %s returned a nil Markdown extension", entry.Descriptor.ID)
			}
			result = append(result, extender)
		}
	}
	if owner, module, ok := p.snapshot.CodeHighlighter(); ok {
		result = append(result, codeHighlighterExtension{owner: owner, module: module, context: ctx})
	}
	return result, nil
}

// postprocess runs active plugin HTML postprocessors in registry order.
func (p *renderPipeline) postprocess(source string, ctx plugin.Context) (string, error) {
	for _, entry := range p.snapshot.Entries {
		for _, module := range entry.Contributions.Postprocessors {
			var err error
			source, err = plugin.Guard(entry.Descriptor.ID, func() (string, error) { return module.Postprocess(ctx, source) })
			if err != nil {
				return "", err
			}
		}
	}
	return source, nil
}

// preprocessMacros protects code using CommonMark's own parser, including long
// fences, blockquote/list fences, and indented code. Registry order decides the
// first matching macro; the central renderer has no knowledge of macro names.
func (p *renderPipeline) preprocessMacros(source string, ctx plugin.Context) (string, []macroInvocation, error) {
	lines := strings.Split(source, "\n")
	protected := codeLines(source)

	var invocations []macroInvocation
	nonce := rand.Text()

	for index, line := range lines {
		if protected[index] {
			continue
		}
		matched := false
		for _, entry := range p.snapshot.Entries {
			for _, macro := range entry.Contributions.Macros {
				type parsed struct {
					arguments plugin.Invocation
					matched   bool
				}
				invocation, err := plugin.Guard(entry.Descriptor.ID, func() (parsed, error) {
					if conditional, ok := macro.(plugin.ConditionalMacro); ok && !conditional.Available(ctx) {
						return parsed{}, nil
					}
					if contextual, ok := macro.(plugin.ContextualMacro); ok {
						args, matched, err := contextual.ParseContext(ctx, line)
						return parsed{args, matched}, err
					}
					args, ok := macro.Parse(line)
					return parsed{args, ok}, nil
				})
				if err != nil {
					return "", nil, err
				}
				if !invocation.matched {
					continue
				}
				placeholder := `<div data-lore-macro="` + nonce + "-" + strconv.Itoa(len(invocations)) + `"></div>`
				invocations = append(invocations, macroInvocation{entry.Descriptor.ID, macro, invocation.arguments, placeholder})
				lines[index] = placeholder
				matched = true
				break
			}
			if matched {
				break
			}
		}
	}

	return strings.Join(lines, "\n"), invocations, nil
}

// expandMacros renders deferred macros and replaces their placeholders in order.
func (p *renderPipeline) expandMacros(source string, invocations []macroInvocation, ctx plugin.Context) (string, error) {
	for _, invocation := range invocations {
		replacement, err := plugin.Guard(invocation.owner, func() (string, error) {
			return invocation.macro.Render(ctx, invocation.arguments)
		})
		if err != nil {
			return "", err
		}
		source = strings.Replace(source, invocation.placeholder, replacement, 1)
	}
	return source, nil
}

// codeLines records code body lines, not fences themselves, whose delimiters
// cannot match a standalone macro invocation.
func codeLines(source string) map[int]bool {
	document := goldmark.New().Parser().Parse(text.NewReader([]byte(source)))
	protected := make(map[int]bool)
	starts := []int{0}

	for index, char := range source {
		if char == '\n' {
			starts = append(starts, index+1)
		}
	}

	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || (node.Kind() != ast.KindFencedCodeBlock && node.Kind() != ast.KindCodeBlock) {
			return ast.WalkContinue, nil
		}
		for i := 0; i < node.Lines().Len(); i++ {
			segment := node.Lines().At(i)
			line := sort.Search(len(starts), func(i int) bool { return starts[i] > segment.Start }) - 1
			protected[line] = true
		}
		return ast.WalkSkipChildren, nil
	})

	return protected
}
