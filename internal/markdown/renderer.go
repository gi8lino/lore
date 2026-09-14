package markdown

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/gi8lino/lore/internal/plugin"
	pluginmarkdown "github.com/gi8lino/lore/plugins/markdown"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	xhtml "golang.org/x/net/html"
)

// Renderer converts Lore Markdown into sanitized HTML.
type Renderer struct {
	// sanitizer removes unsafe HTML from rendered output.
	sanitizer *bluemonday.Policy
	registry  *plugin.Registry
	manager   *plugin.Manager
}

// Heading describes one rendered Markdown heading used in a page table of contents.
type Heading struct {
	// Level is the HTML heading level from 1 through 6.
	Level int
	// ID is the rendered heading anchor identifier.
	ID string
	// Title is the plain-text heading label.
	Title string
}

// RenderedPage contains sanitized page HTML and its extracted heading structure.
type RenderedPage struct {
	// HTML is the sanitized rendered Markdown.
	HTML string
	// Contents contains headings in document order.
	Contents []Heading
}

// Functions supplies request-local macro capabilities and variable provenance.
// Bindings cannot activate an unregistered macro.
type Functions struct {
	Capabilities map[string]plugin.Capability
	Context      context.Context
	Variables    []Variable
	Macros       map[string]plugin.MacroRenderer
}

// Close releases the owned plugin runtime. Explicit-registry renderers leave
// ownership with their caller.
func (r *Renderer) Close(ctx context.Context) error {
	if r.manager != nil {
		return r.manager.Close(ctx)
	}
	return nil
}

// NewWithRegistry uses an application-owned registry for every render path.
// An empty registry enables only the remaining core Markdown features.
func NewWithRegistry(registry *plugin.Registry) *Renderer {
	if registry == nil {
		registry = &plugin.Registry{}
	}

	return &Renderer{sanitizer: newSanitizer(), registry: registry}
}

// engine constructs a Goldmark renderer from administrator-controlled options.
func engine(options Options, contributed []goldmark.Extender, ranges ...variableRange) goldmark.Markdown {
	extensions := make([]goldmark.Extender, 0, 8)

	if options.Autolinks {
		extensions = append(extensions, extension.Linkify)
	}
	if options.Footnotes {
		extensions = append(extensions, extension.Footnote)
	}
	if options.DefinitionLists {
		extensions = append(extensions, extension.DefinitionList)
	}
	if options.Typographer {
		var typographer goldmark.Extender = extension.Typographer
		if options.CodingLigatures {
			typographer = extension.NewTypographer(
				extension.WithTypographicSubstitutions(
					extension.TypographicSubstitutions{
						extension.EnDash:          nil,
						extension.EmDash:          nil,
						extension.LeftAngleQuote:  nil,
						extension.RightAngleQuote: nil,
					},
				),
			)
		}
		extensions = append(extensions, typographer)
	}
	if options.SyntaxHighlighting {
		extensions = append(
			extensions,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github-dark"),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		)
	}

	extensions = append(extensions, contributed...)
	return goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.Prioritized(imageWidthTransformer{}, 100),
				util.Prioritized(
					variableTransformer{ranges: ranges},
					200,
				),
			),
		),
		goldmark.WithRendererOptions(
			goldhtml.WithUnsafe(),
			renderer.WithNodeRenderers(
				util.Prioritized(taskCheckBoxRenderer{}, 100),
				util.Prioritized(
					variableNodeRenderer{ranges: ranges},
					100,
				),
			),
		),
	)
}

// Links extracts unique canonical wiki-link targets from Markdown source.
func Links(source string) []string {
	seen := make(map[string]bool)
	var links []string

	walkWikiLinks(source, func(target, _ string) {
		pageTarget, _ := SplitHeadingTarget(target)
		slug := Slug(pageTarget)
		if slug == "" || seen[slug] {
			return
		}

		seen[slug] = true
		links = append(links, slug)
	})
	return links
}

// Render converts Markdown into sanitized HTML using default rendering options.
func (r *Renderer) Render(source string) (string, error) {
	return r.RenderResolvedWithOptions(source, Slug, DefaultOptions())
}

// RenderResolved converts Markdown into sanitized HTML using default rendering options and a custom wiki-link resolver.
func (r *Renderer) RenderResolved(
	source string,
	resolve func(string) string,
) (string, error) {
	return r.RenderResolvedWithOptions(source, resolve, DefaultOptions())
}

// RenderResolvedWithOptions converts Markdown using administrator-controlled rendering options.
func (r *Renderer) RenderResolvedWithOptions(
	source string,
	resolve func(string) string,
	options Options,
) (string, error) {
	rendered, err := r.RenderPageResolvedWithOptions(
		source,
		resolve,
		options,
	)
	if err != nil {
		return "", err
	}

	return rendered.HTML, nil
}

// RenderPageResolved renders Markdown using default options and returns both HTML and page contents.
func (r *Renderer) RenderPageResolved(
	source string,
	resolve func(string) string,
) (RenderedPage, error) {
	return r.RenderPageResolvedWithOptions(
		source,
		resolve,
		DefaultOptions(),
	)
}

// RenderPageResolvedWithOptions renders Markdown and returns both sanitized HTML and page contents.
func (r *Renderer) RenderPageResolvedWithOptions(
	source string,
	resolve func(string) string,
	options Options,
) (RenderedPage, error) {
	return r.RenderPageResolvedWithFunctions(
		source,
		resolve,
		options,
		Functions{},
	)
}

// RenderPageResolvedWithFunctions renders Markdown and expands trusted dynamic page functions.
func (r *Renderer) RenderPageResolvedWithFunctions(
	source string,
	resolve func(string) string,
	options Options,
	functions Functions,
) (RenderedPage, error) {
	execution := functions.Context
	if execution == nil {
		execution = context.Background()
	}
	execution, cancel := context.WithTimeout(execution, 30*time.Second)
	defer cancel()
	functions.Context = execution
	snapshot, release := r.registry.Acquire()
	defer release()
	options.CodingLigatures = options.CodingLigatures || snapshot.HasRenderPolicy("coding-ligatures")
	options.pipeline = newRenderPipeline(snapshot, r.pluginFeatures(options), functions)
	if len(functions.Variables) != 0 {
		return r.renderPageWithVariables(
			source,
			resolve,
			options,
			functions,
		)
	}

	return r.renderPage(source, resolve, options)
}

// renderPageWithVariables annotates expanded variable origins without changing the rendered document.
func (r *Renderer) renderPageWithVariables(
	source string,
	resolve func(string) string,
	options Options,
	functions Functions,
) (RenderedPage, error) {
	plain, _ := resolveVariableTokens(source, functions.Variables)

	normal, err := r.renderPage(
		plain,
		resolve,
		options,
	)
	if err != nil {
		return RenderedPage{}, err
	}

	options.variables = functions.Variables

	annotated, err := r.renderPage(
		source,
		resolve,
		options,
	)
	if err != nil {
		return normal, nil
	}

	if equivalentVariableHTML(annotated.HTML, normal.HTML) {
		normal.HTML = annotated.HTML
	}

	return normal, nil
}

// renderPage renders one Markdown page with optional dynamic functions and heading extraction.
func (r *Renderer) renderPage(
	source string,
	resolve func(string) string,
	options Options,
) (RenderedPage, error) {
	source, invocations, err := options.pipeline.preprocessMacros(source, r.moduleContext(resolve, options))
	if err != nil {
		return RenderedPage{}, err
	}
	raw, err := r.renderRawResolved(source, resolve, options)
	if err != nil {
		return RenderedPage{}, err
	}
	// Preserve the established contents list: generated macro headings are not
	// part of the source page's navigation.
	contents := extractHeadings(r.sanitizer.Sanitize(raw))
	raw, err = options.pipeline.expandMacros(raw, invocations, r.moduleContext(resolve, options))
	if err != nil {
		return RenderedPage{}, err
	}
	raw, err = options.pipeline.postprocess(raw, r.moduleContext(resolve, options))
	if err != nil {
		return RenderedPage{}, err
	}
	return RenderedPage{HTML: r.sanitizer.Sanitize(raw), Contents: contents}, nil
}

// renderRawResolved renders Markdown extensions into unsanitized HTML for recursive block rendering.
func (r *Renderer) renderRawResolved(
	source string,
	resolve func(string) string,
	options Options,
) (string, error) {
	if err := options.pipeline.context.Err(); err != nil {
		return "", err
	}
	if options.depth >= 64 {
		return "", errors.New("markdown nesting limit exceeded")
	}
	options.depth++
	var err error

	ctx := r.moduleContext(resolve, options)
	source, err = options.pipeline.preprocess(source, ctx)
	if err != nil {
		return "", err
	}

	if options.WikiLinks {
		source = rewriteWikiLinks(
			source,
			resolve,
			wikiLinkPrefix(options),
		)
	}

	source, ranges := resolveVariableTokens(
		source,
		options.variables,
	)

	var output bytes.Buffer

	extensions, err := options.pipeline.extensions(ctx)
	if err != nil {
		return "", err
	}
	// Conversion invokes contributed parsers, transformers, and node renderers.
	_, err = plugin.Guard("Markdown conversion", func() (struct{}, error) {
		return struct{}{}, engine(options, extensions, ranges...).Convert([]byte(source), &output)
	})
	if err != nil {
		return "", err
	}

	raw := output.String()

	return raw, nil
}

// fenceDelimiter returns the Markdown fence marker when a line starts a fenced code block.
func fenceDelimiter(line string) string { return pluginmarkdown.Fence(line) }

// walkWikiLinks visits wiki links outside fenced code blocks in source order.
func walkWikiLinks(
	source string,
	visit func(target, label string),
) {
	fence := ""

	for line := range strings.SplitSeq(source, "\n") {
		marker := fenceDelimiter(line)

		if fence != "" {
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}

			continue
		}

		if marker != "" {
			fence = marker
			continue
		}

		walkWikiLinksLine(line, visit)
	}
}

// walkWikiLinksLine visits syntactically valid wiki links in one Markdown line.
func walkWikiLinksLine(
	line string,
	visit func(target, label string),
) {
	for offset := 0; offset < len(line); {
		start := strings.Index(line[offset:], "[[")

		if start < 0 {
			return
		}

		start += offset

		if start > 0 && line[start-1] == '\\' {
			offset = start + 2
			continue
		}

		end := strings.Index(line[start+2:], "]]")
		if end < 0 {
			return
		}

		end += start + 2

		target, label, ok := parseWikiLink(
			line[start+2 : end],
		)

		if ok {
			visit(target, label)
		}

		offset = end + 2
	}
}

// parseWikiLink splits a wiki-link body into target and optional label.
func parseWikiLink(
	value string,
) (target string, label string, ok bool) {
	target, label, hasLabel := strings.Cut(value, "|")
	target = strings.TrimSpace(target)

	if target == "" {
		return "", "", false
	}

	label = strings.TrimSpace(label)

	if !hasLabel || label == "" {
		label = target
	}

	return target, label, true
}

// SplitHeadingTarget separates a wiki page target from an optional heading fragment.
func SplitHeadingTarget(target string) (page string, heading string) {
	page, heading, _ = strings.Cut(strings.TrimSpace(target), "#")
	return strings.TrimSpace(page), strings.TrimSpace(heading)
}

// HeadingID converts a human-readable heading reference into Lore's heading anchor form.
func HeadingID(value string) string {
	return Slug(strings.ReplaceAll(value, "/", " "))
}

// wikiLinkPrefix returns the configured wiki-link URL prefix.
func wikiLinkPrefix(options Options) string {
	if strings.TrimSpace(options.WikiLinkPrefix) == "" {
		return "/pages/"
	}

	return options.WikiLinkPrefix
}

// rewriteWikiLinks converts wiki-link syntax outside fenced code blocks into Markdown links.
func rewriteWikiLinks(
	source string,
	resolve func(string) string,
	prefix string,
) string {
	lines := strings.Split(source, "\n")
	fence := ""

	for index, line := range lines {
		marker := fenceDelimiter(line)

		if fence != "" {
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}

			continue
		}

		if marker != "" {
			fence = marker
			continue
		}

		lines[index] = rewriteWikiLinksLine(
			line,
			resolve,
			prefix,
		)
	}

	return strings.Join(lines, "\n")
}

// rewriteWikiLinksLine converts wiki links in one Markdown line without regular expressions.
func rewriteWikiLinksLine(
	line string,
	resolve func(string) string,
	prefix string,
) string {
	var output strings.Builder

	offset := 0

	for offset < len(line) {
		start := strings.Index(line[offset:], "[[")

		if start < 0 {
			output.WriteString(line[offset:])
			break
		}

		start += offset

		if start > 0 && line[start-1] == '\\' {
			output.WriteString(line[offset : start+2])

			offset = start + 2
			continue
		}

		end := strings.Index(line[start+2:], "]]")
		if end < 0 {
			output.WriteString(line[offset:])
			break
		}

		end += start + 2

		target, label, ok := parseWikiLink(
			line[start+2 : end],
		)

		if !ok {
			output.WriteString(line[offset : end+2])

			offset = end + 2
			continue
		}

		output.WriteString(line[offset:start])
		output.WriteByte('[')
		output.WriteString(label)
		output.WriteString("](")
		pageTarget, heading := SplitHeadingTarget(target)
		resolved := resolve(pageTarget)
		output.WriteString(prefix)
		output.WriteString(resolved)
		if heading != "" {
			output.WriteByte('#')
			output.WriteString(HeadingID(heading))
		}
		output.WriteByte(')')

		offset = end + 2
	}

	return output.String()
}

// htmlHeadingLevel returns the numeric level of an h1-h6 element.
func htmlHeadingLevel(
	node *xhtml.Node,
) (level int, ok bool) {
	if node.Type != xhtml.ElementNode ||
		len(node.Data) != 2 {
		return 0, false
	}

	if node.Data[0] != 'h' {
		return 0, false
	}

	digit := node.Data[1]

	if digit < '1' || digit > '6' {
		return 0, false
	}

	return int(digit - '0'), true
}

// extractHeadings extracts rendered heading IDs and labels for page navigation.
func extractHeadings(rendered string) []Heading {
	document, err := xhtml.Parse(
		strings.NewReader(rendered),
	)
	if err != nil {
		return nil
	}

	var contents []Heading

	var walk func(*xhtml.Node)

	walk = func(node *xhtml.Node) {
		if level, ok := htmlHeadingLevel(node); ok {
			id := htmlAttribute(node, "id")

			if id != "" {
				contents = append(
					contents,
					Heading{
						Level: level,
						ID:    id,
						Title: strings.TrimSpace(
							htmlText(node),
						),
					},
				)
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(document)

	return contents
}

// htmlAttribute returns one HTML node attribute by key.
func htmlAttribute(
	node *xhtml.Node,
	key string,
) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}

	return ""
}

// htmlText returns the concatenated text content below an HTML node.
func htmlText(node *xhtml.Node) string {
	var output strings.Builder

	var walk func(*xhtml.Node)

	walk = func(current *xhtml.Node) {
		if current.Type == xhtml.TextNode {
			output.WriteString(current.Data)
		}

		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(node)

	return output.String()
}
