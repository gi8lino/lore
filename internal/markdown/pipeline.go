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
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// renderPipeline pins an active contribution set for the whole document,
// including nested Markdown and the variable-provenance rendering pass.
type renderPipeline struct {
	context  context.Context
	snapshot plugin.Snapshot
	features map[string]bool
	macros   map[string]plugin.MacroRenderer
}

type macroInvocation struct {
	owner       string
	macro       plugin.Macro
	arguments   plugin.Invocation
	placeholder string
}

func newRenderPipeline(snapshot plugin.Snapshot, options Options, functions Functions) *renderPipeline {
	return &renderPipeline{context: functions.Context, snapshot: snapshot, features: moduleFeatures(options), macros: maps.Clone(functions.Macros)}
}

func (r *Renderer) moduleContext(resolve func(string) string, options Options) plugin.Context {
	return plugin.Context{
		Context:        options.pipeline.context,
		Features:       maps.Clone(options.pipeline.features),
		Macros:         maps.Clone(options.pipeline.macros),
		RenderMarkdown: func(source string) (string, error) { return r.renderRawResolved(source, resolve, options) },
	}
}

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
	return result, nil
}

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
