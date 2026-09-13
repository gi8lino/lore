package markdown

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"context"
	"github.com/gi8lino/lore/internal/icons"
	"github.com/gi8lino/lore/internal/navigation"
	"github.com/gi8lino/lore/internal/plugin"
	"github.com/gi8lino/lore/internal/plugincap"
	"github.com/gi8lino/lore/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type preprocessorFunc func(plugin.Context, string) (string, error)

func (f preprocessorFunc) Preprocess(c plugin.Context, s string) (string, error) { return f(c, s) }

type postprocessorFunc func(plugin.Context, string) (string, error)

func (f postprocessorFunc) Postprocess(c plugin.Context, s string) (string, error) { return f(c, s) }

type extensionFunc func(plugin.Context) goldmark.Extender

func (f extensionFunc) Extension(c plugin.Context) goldmark.Extender { return f(c) }

type testMacro struct {
	parse  func(string) (plugin.Invocation, bool)
	render func(plugin.Context, plugin.Invocation) (string, error)
}

func (m testMacro) Name() string                             { return "status" }
func (m testMacro) Parse(s string) (plugin.Invocation, bool) { return m.parse(s) }
func (m testMacro) Render(c plugin.Context, v plugin.Invocation) (string, error) {
	return m.render(c, v)
}
func statusMacro(html string) testMacro {
	return testMacro{
		parse:  func(s string) (plugin.Invocation, bool) { return []byte(`{}`), strings.TrimSpace(s) == "{{status}}" },
		render: func(plugin.Context, plugin.Invocation) (string, error) { return html, nil },
	}
}

func TestPluginStagesAndMacroOutputShareFinalSanitizer(t *testing.T) {
	registry := &plugin.Registry{}
	require.NoError(t, registry.Register(plugin.Descriptor{ID: "custom", Name: "Custom"}, plugin.Contributions{
		Preprocessors: []plugin.Preprocessor{preprocessorFunc(func(_ plugin.Context, source string) (string, error) {
			return strings.ReplaceAll(source, "INPUT", "~~processed~~") + `<script>pre()</script>`, nil
		})},
		MarkdownExtensions: []plugin.MarkdownExtension{extensionFunc(func(plugin.Context) goldmark.Extender { return extension.Strikethrough })},
		Macros:             []plugin.Macro{statusMacro(`<div class="status" onclick="bad()">macro<script>macro()</script><a href="javascript:bad()">link</a></div>`)},
		Postprocessors: []plugin.Postprocessor{postprocessorFunc(func(_ plugin.Context, html string) (string, error) {
			require.Contains(t, html, `class="status"`)
			require.Contains(t, html, "<del>processed</del>")
			return html + `<p>post</p><script>post()</script><img src="/ok.png" onerror="bad()">`, nil
		})},
	}))
	rendered, err := NewWithRegistry(registry).RenderResolvedWithOptions("INPUT\n\n{{status}}\n\n", Slug, Options{})
	require.NoError(t, err)
	assert.Contains(t, rendered, "<del>processed</del>")
	assert.Contains(t, rendered, `class="status"`)
	assert.Contains(t, rendered, "<p>post</p>")
	assert.NotContains(t, rendered, "<script")
	assert.NotContains(t, rendered, "onclick")
	assert.NotContains(t, rendered, "onerror")
	assert.NotContains(t, rendered, "javascript:")
}

func TestMacroCodeBoundaries(t *testing.T) {
	registry := &plugin.Registry{}
	require.NoError(t, registry.Register(plugin.Descriptor{ID: "test", Name: "Test"}, plugin.Contributions{Macros: []plugin.Macro{statusMacro("EXPANDED")}}))
	for _, source := range []string{
		"```\n{{status}}\n```",
		"~~~~\n~~~\n{{status}}\n~~~~~",
		"````markdown\n```\n{{status}}\n`````",
		"```\n```not-a-close\n{{status}}\n```",
		"> ```\n> {{status}}\n> ```",
		"- item\n\n  ```\n  {{status}}\n  ```",
		"    {{status}}",
		"~~~\n{{status}}",
	} {
		t.Run(source, func(t *testing.T) {
			got, err := NewWithRegistry(registry).Render(source)
			require.NoError(t, err)
			assert.NotContains(t, got, "EXPANDED")
			assert.Contains(t, got, "{{status}}")
		})
	}
}

func TestCalloutsCanBeRemovedAndRegisteredWithoutReplacingRenderer(t *testing.T) {
	renderer := testRenderer(t)
	registry := renderer.registry
	var callouts plugin.Entry
	for _, entry := range registry.Snapshot().Entries {
		if entry.Descriptor.ID == "io.lore.callouts" {
			callouts = entry
		}
	}
	source := "!!! warning\nSee **this** and [[Page]].\n"
	got, err := renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `class="callout warning"`)
	assert.Contains(t, got, "<strong>this</strong>")
	assert.Contains(t, got, `href="/pages/page"`)
	require.NoError(t, registry.Unregister("io.lore.callouts"))
	got, err = renderer.Render(source)
	require.NoError(t, err)
	assert.NotContains(t, got, `class="callout`)
	assert.Contains(t, got, "!!! warning")
	require.NoError(t, registry.Register(callouts.Descriptor, callouts.Contributions))
	got, err = renderer.Render(source)
	require.NoError(t, err)
	assert.Contains(t, got, `class="callout warning"`)
	options := DefaultOptions()
	options.Callouts = false
	got, err = renderer.RenderResolvedWithOptions(source, Slug, options)
	require.NoError(t, err)
	assert.NotContains(t, got, `class="callout`)
}

func TestRenderSnapshotSurvivesRemovalDuringNestedRender(t *testing.T) {
	renderer := testRenderer(t)
	registry := renderer.registry
	var once sync.Once
	require.NoError(t, registry.Register(plugin.Descriptor{ID: "remove", Name: "Remove"}, plugin.Contributions{
		Preprocessors: []plugin.Preprocessor{preprocessorFunc(func(_ plugin.Context, source string) (string, error) {
			once.Do(func() { require.NoError(t, registry.Unregister("io.lore.callouts")) })
			return source, nil
		})},
	}))
	got, err := renderer.Render("!!! warning\n!!! note\nNested\n")
	require.NoError(t, err)
	assert.Contains(t, got, `class="callout warning"`)
	assert.Contains(t, got, `class="callout note"`)
	got, err = renderer.Render("!!! note\nNext\n")
	require.NoError(t, err)
	assert.NotContains(t, got, `class="callout`)
}

func TestModulePanicsAndErrorsReturnRenderErrors(t *testing.T) {
	failure := errors.New("module failed")
	panicking := statusMacro("")
	panicking.parse = func(string) (plugin.Invocation, bool) { panic("parse") }
	rendering := statusMacro("")
	rendering.render = func(plugin.Context, plugin.Invocation) (string, error) { panic("render") }
	for name, modules := range map[string]plugin.Contributions{
		"pre":       {Preprocessors: []plugin.Preprocessor{preprocessorFunc(func(plugin.Context, string) (string, error) { panic("pre") })}},
		"post":      {Postprocessors: []plugin.Postprocessor{postprocessorFunc(func(plugin.Context, string) (string, error) { panic("post") })}},
		"extension": {MarkdownExtensions: []plugin.MarkdownExtension{extensionFunc(func(plugin.Context) goldmark.Extender { panic("extension") })}},
		"parse":     {Macros: []plugin.Macro{panicking}},
		"render":    {Macros: []plugin.Macro{rendering}},
		"error":     {Postprocessors: []plugin.Postprocessor{postprocessorFunc(func(plugin.Context, string) (string, error) { return "", failure })}},
	} {
		t.Run(name, func(t *testing.T) {
			registry := &plugin.Registry{}
			require.NoError(t, registry.Register(plugin.Descriptor{ID: name, Name: name}, modules))
			got, err := NewWithRegistry(registry).Render("{{status}}")
			require.Error(t, err)
			assert.Empty(t, got)
			assert.Contains(t, err.Error(), name)
			if name == "error" {
				assert.ErrorIs(t, err, failure)
			}
		})
	}
}

func TestModuleRecursionIsBounded(t *testing.T) {
	registry := &plugin.Registry{}
	require.NoError(t, registry.Register(plugin.Descriptor{ID: "loop", Name: "Loop"}, plugin.Contributions{
		Preprocessors: []plugin.Preprocessor{preprocessorFunc(func(ctx plugin.Context, source string) (string, error) { return ctx.RenderMarkdown(source) })},
	}))
	_, err := NewWithRegistry(registry).Render("loop")
	require.ErrorContains(t, err, "nesting limit")
}

func TestRemovedMacrosCannotBeActivatedByRequestBindings(t *testing.T) {
	renderer := testRenderer(t)
	registry := renderer.registry
	require.NoError(t, registry.Unregister("io.lore.subpages"))
	got, err := NewWithRegistry(registry).RenderPageResolvedWithFunctions("{{subpages}}", Slug, DefaultOptions(), Functions{
		Macros: map[string]plugin.MacroRenderer{"subpages": func(plugin.Invocation) (string, error) { t.Fatal("removed macro invoked"); return "", nil }},
	})
	require.NoError(t, err)
	assert.Contains(t, got.HTML, "{{subpages}}")
}

func TestSubpagesMarkupAndStaticIconsSurviveCentralSanitizer(t *testing.T) {
	nodes := plugincap.Navigation([]navigation.Node{{Slug: "child", Title: "Child", Page: true, Icon: "book-lucide"}}, func(s string) string { return "/pages/" + s })
	got, err := testRenderer(t).RenderPageResolvedWithFunctions("{{subpages}}", Slug, DefaultOptions(), Functions{
		Capabilities: plugincap.Capabilities(nil, nodes),
	})
	require.NoError(t, err)
	assert.Contains(t, got.HTML, `<nav class="subpage-toc"`)
	assert.Contains(t, got.HTML, `class="subpage-toc-link"`)
	assert.Contains(t, got.HTML, `class="subpage-toc-list subpage-toc-root"`)
	assert.Contains(t, got.HTML, `<svg`)
	assert.Contains(t, got.HTML, `viewbox="0 0 24 24"`)
	assert.Contains(t, got.HTML, `<path`)
	assert.Contains(t, got.HTML, `stroke="currentColor"`)
	assert.Empty(t, got.Contents)
	svg := string(icons.SVG("github-simple", 17))
	assert.Contains(t, newSanitizer().Sanitize(svg), `<path`)
}

func TestSanitizerRejectsActiveSVGAndUnsafeMacroMarkup(t *testing.T) {
	source := `<svg onload="alert(1)"><script>alert(1)</script><foreignObject><iframe src="https://evil.test"></iframe></foreignObject><use href="https://evil.test/icon.svg#x"></use><animate attributeName="href" values="javascript:alert(1)"></animate><path d="M0 0h1" fill="url(https://evil.test/paint)" style="fill:url(https://evil.test)"></path></svg>`
	got := newSanitizer().Sanitize(source)
	for _, denied := range []string{"onload", "<script", "<foreignObject", "<iframe", "<use", "<animate", "javascript:", "evil.test", "style="} {
		assert.NotContains(t, got, denied)
	}
	assert.Contains(t, got, `<path d="M0 0h1"`)
}

// Native Goldmark adapters can register AST transformers and node renderers;
// their generated HTML still traverses the same final sanitizer.
type testASTExtension struct{}

func (testASTExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithASTTransformers(util.Prioritized(testTransformer{}, 500)))
	m.Renderer().AddOptions(gmrenderer.WithNodeRenderers(util.Prioritized(testNodeRenderer{}, 50)))
}

type testTransformer struct{}

func (testTransformer) Transform(node *ast.Document, _ text.Reader, _ parser.Context) {
	node.FirstChild().AppendChild(node.FirstChild(), ast.NewString([]byte("AST")))
}

type testNodeRenderer struct{}

func (testNodeRenderer) RegisterFuncs(r gmrenderer.NodeRendererFuncRegisterer) {
	r.Register(ast.KindString, func(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString(`<strong>AST</strong><script>node()</script>`)
		}
		return ast.WalkContinue, nil
	})
}
func TestModuleContributesASTAndNodeRenderer(t *testing.T) {
	registry := &plugin.Registry{}
	require.NoError(t, registry.Register(plugin.Descriptor{ID: "ast", Name: "AST"}, plugin.Contributions{MarkdownExtensions: []plugin.MarkdownExtension{extensionFunc(func(plugin.Context) goldmark.Extender { return testASTExtension{} })}}))
	got, err := NewWithRegistry(registry).Render("Text")
	require.NoError(t, err)
	assert.Contains(t, got, "Text<strong>AST</strong>")
	assert.NotContains(t, got, "<script")
}

func TestUnavailableMacroRemainsOrdinaryMarkdown(t *testing.T) {
	source := "Before\n{{pages query=\"status:verified\"}}\nAfter"
	renderer := testRenderer(t)
	got, err := renderer.Render(source)
	require.NoError(t, err)
	// Compare with a renderer lacking the macro. No placeholder should split
	// this paragraph or cause another pass through preprocessors.
	expected, err := NewWithRegistry(&plugin.Registry{}).Render(source)
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestMacroCapabilitiesStayRequestLocal(t *testing.T) {
	renderer := testRenderer(t)
	var wg sync.WaitGroup
	for _, title := range []string{"First", "Second"} {
		wg.Go(func() {
			got, err := renderer.RenderPageResolvedWithFunctions("{{subpages}}", Slug, DefaultOptions(), Functions{
				Capabilities: plugincap.Capabilities(nil, []pluginapi.NavigationNode{{Title: title, Page: true, URL: "/pages/child"}}),
			})
			assert.NoError(t, err)
			assert.Contains(t, got.HTML, ">"+title+"</span>")
		})
	}
	wg.Wait()
	got, err := renderer.Render("{{subpages}}")
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(got))
}

func testRenderer(t testing.TB) *Renderer {
	t.Helper()
	renderer, err := New(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = renderer.Close(context.Background()) })
	return renderer
}
