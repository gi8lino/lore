package markdown

import (
	"bytes"
	stdhtml "html"
	"strconv"
	"strings"
	"unicode"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/gi8lino/lore/internal/ascii"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Renderer converts wiki Markdown into sanitized HTML.
type Renderer struct {
	// sanitizer removes unsafe HTML from rendered output.
	sanitizer *bluemonday.Policy
}

// Options controls optional Markdown rendering features.
type Options struct {
	// WikiLinks enables [[Wiki Link]] resolution.
	WikiLinks bool
	// WikiLinkPrefix is prepended to resolved wiki-link targets. Empty uses /pages/.
	WikiLinkPrefix string
	// Callouts enables Lore callout blocks.
	Callouts bool
	// Tabs enables Material-style tab blocks.
	Tabs bool
	// Details enables collapsible detail blocks.
	Details bool
	// Tables enables GitHub-flavored Markdown tables.
	Tables bool
	// TableStyles enables trusted theme-aware table colors.
	TableStyles bool
	// TableSorting enables client-side sorting for opted-in tables.
	TableSorting bool
	// TableFiltering enables client-side filtering for opted-in tables.
	TableFiltering bool
	// Strikethrough enables GitHub-flavored strikethrough.
	Strikethrough bool
	// TaskLists enables GitHub-flavored task lists.
	TaskLists bool
	// Autolinks enables automatic URL and email links.
	Autolinks bool
	// SyntaxHighlighting enables server-side fenced-code highlighting.
	SyntaxHighlighting bool
	// Footnotes enables Markdown footnotes.
	Footnotes bool
	// DefinitionLists enables Markdown definition lists.
	DefinitionLists bool
	// Typographer enables typographic punctuation substitutions.
	Typographer bool
}

// DefaultOptions returns the rendering behavior used before administrator customization.
func DefaultOptions() Options {
	return Options{
		Autolinks:          true,
		Callouts:           true,
		DefinitionLists:    true,
		Details:            true,
		Footnotes:          true,
		Strikethrough:      true,
		SyntaxHighlighting: true,
		TableFiltering:     true,
		Tables:             true,
		TableSorting:       true,
		TableStyles:        true,
		Tabs:               true,
		TaskLists:          true,
		Typographer:        true,
		WikiLinks:          true,
		WikiLinkPrefix:     "/pages/",
	}
}

// tableStyle describes trusted presentation classes applied to one rendered table.
type tableStyle struct {
	// header is the optional header-row tone.
	header string
	// rows maps one-based body rows to tones.
	rows map[int]string
	// columns maps one-based columns to tones.
	columns map[int]string
	// cells maps one-based body-row and column positions to tones.
	cells map[[2]int]string
	// sortable marks the table for client-side column sorting.
	sortable bool
	// filterable marks the table for client-side row filtering.
	filterable bool
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

// Functions contains trusted dynamic HTML for opt-in Markdown functions.
type Functions struct {
	// Subpages is the generated navigation tree inserted by {{subpages}}.
	Subpages string
}

const subpagesPlaceholder = `<div class="lore-function-subpages"></div>`

// tabSection contains one parsed Markdown tab label and body.
type tabSection struct {
	// title is the visible tab label.
	title string
	// body is the de-indented Markdown body for the tab.
	body string
}

// New constructs the package default implementation.
func New() *Renderer {
	policy := bluemonday.UGCPolicy()

	policy.AllowElements("div", "button", "details", "summary")
	policy.AllowAttrs("class").
		OnElements("aside", "pre", "code", "span", "div", "button", "details", "summary", "table", "thead", "tbody", "tr", "th", "td")
	policy.AllowAttrs("role").OnElements("div", "button")
	policy.AllowAttrs("type", "aria-selected").OnElements("button")
	policy.AllowAttrs("open").OnElements("details")
	policy.AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")

	return &Renderer{sanitizer: policy}
}

// engine constructs a Goldmark renderer from administrator-controlled options.
func engine(options Options) goldmark.Markdown {
	extensions := make([]goldmark.Extender, 0, 8)

	if options.Tables {
		extensions = append(extensions, extension.Table)
	}
	if options.Strikethrough {
		extensions = append(extensions, extension.Strikethrough)
	}
	if options.TaskLists {
		extensions = append(extensions, extension.TaskList)
	}
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
		extensions = append(extensions, extension.Typographer)
	}
	if options.SyntaxHighlighting {
		extensions = append(extensions, highlighting.NewHighlighting(
			highlighting.WithStyle("github-dark"),
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
		))
	}

	return goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(goldhtml.WithUnsafe()),
	)
}

// Slug converts human-readable page text into a canonical wiki slug.
func Slug(value string) string {
	value = strings.TrimSpace(value)
	var output strings.Builder
	separator := false

	for _, r := range value {
		r = unicode.ToLower(r)
		switch {
		case isSlugRune(r):
			if separator && output.Len() > 0 {
				output.WriteByte('-')
			}

			separator = false

			output.WriteRune(r)
		default:
			separator = true
		}
	}

	return strings.Trim(output.String(), "-")
}

// isSlugRune reports whether character can be preserved in a canonical wiki slug.
func isSlugRune(character rune) bool {
	return ascii.IsAlphanumeric(character) || character == '/' || character == '_' || character == '-'
}

// Links extracts unique canonical wiki-link targets from Markdown source.
func Links(source string) []string {
	seen := make(map[string]bool)
	var links []string

	walkWikiLinks(source, func(target, _ string) {
		slug := Slug(target)
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
func (r *Renderer) RenderResolved(source string, resolve func(string) string) (string, error) {
	return r.RenderResolvedWithOptions(source, resolve, DefaultOptions())
}

// RenderResolvedWithOptions converts Markdown using administrator-controlled rendering options.
func (r *Renderer) RenderResolvedWithOptions(
	source string,
	resolve func(string) string,
	options Options,
) (string, error) {
	rendered, err := r.RenderPageResolvedWithOptions(source, resolve, options)
	if err != nil {
		return "", err
	}

	return rendered.HTML, nil
}

// RenderPageResolved renders Markdown using default options and returns both HTML and page contents.
func (r *Renderer) RenderPageResolved(source string, resolve func(string) string) (RenderedPage, error) {
	return r.RenderPageResolvedWithOptions(source, resolve, DefaultOptions())
}

// RenderPageResolvedWithOptions renders Markdown and returns both sanitized HTML and page contents.
func (r *Renderer) RenderPageResolvedWithOptions(
	source string,
	resolve func(string) string,
	options Options,
) (RenderedPage, error) {
	return r.RenderPageResolvedWithFunctions(source, resolve, options, Functions{})
}

// RenderPageResolvedWithFunctions renders Markdown and expands trusted dynamic page functions.
func (r *Renderer) RenderPageResolvedWithFunctions(
	source string,
	resolve func(string) string,
	options Options,
	functions Functions,
) (RenderedPage, error) {
	source = preprocessFunctions(source)
	raw, err := r.renderRawResolved(source, resolve, options)
	if err != nil {
		return RenderedPage{}, err
	}

	html := r.sanitizer.Sanitize(raw)
	contents := extractHeadings(html)
	html = strings.ReplaceAll(html, subpagesPlaceholder, functions.Subpages)

	return RenderedPage{HTML: html, Contents: contents}, nil
}

// preprocessFunctions replaces standalone function calls outside fenced code with safe placeholders.
func preprocessFunctions(source string) string {
	lines := strings.Split(source, "\n")
	output := make([]string, 0, len(lines))

	for index := 0; index < len(lines); {
		if marker := fenceDelimiter(lines[index]); marker != "" {
			index = appendFencedBlock(lines, index, marker, &output)
			continue
		}

		if strings.TrimSpace(lines[index]) == "{{subpages}}" {
			output = append(output, subpagesPlaceholder)
		} else {
			output = append(output, lines[index])
		}

		index++
	}

	return strings.Join(output, "\n")
}

// renderRawResolved renders Markdown extensions into unsanitized HTML for recursive block rendering.
func (r *Renderer) renderRawResolved(source string, resolve func(string) string, options Options) (string, error) {
	var err error

	if options.Tabs {
		source, err = r.preprocessTabs(source, resolve, options)
		if err != nil {
			return "", err
		}
	}
	if options.Details {
		source, err = r.preprocessDetails(source, resolve, options)
		if err != nil {
			return "", err
		}
	}
	if options.Callouts {
		source, err = r.preprocessCallouts(source, resolve, options)
		if err != nil {
			return "", err
		}
	}
	if options.WikiLinks {
		source = rewriteWikiLinks(source, resolve, wikiLinkPrefix(options))
	}
	if tableDirectivesEnabled(options) {
		source = preprocessTableDirectives(source, options)
	}

	var output bytes.Buffer
	if err := engine(options).Convert([]byte(source), &output); err != nil {
		return "", err
	}

	raw := output.String()

	if tableDirectivesEnabled(options) {
		raw, err = applyTableDirectiveMarkers(raw, options)
		if err != nil {
			return "", err
		}
	}

	return raw, nil
}

// preprocessTabs converts consecutive Material-style tab blocks into semantic tab markup.
func (r *Renderer) preprocessTabs(source string, resolve func(string) string, options Options) (string, error) {
	lines := strings.Split(source, "\n")
	out := make([]string, 0, len(lines))

	for index := 0; index < len(lines); {
		if marker := fenceDelimiter(lines[index]); marker != "" {
			index = appendFencedBlock(lines, index, marker, &out)
			continue
		}

		title, ok := parseTabTitle(lines[index])
		if !ok {
			out = append(out, lines[index])

			index++
			continue
		}

		var sections []tabSection

		for ok {
			bodyLines, next := indentedBody(lines, index+1)
			sections = append(sections, tabSection{title: title, body: strings.Join(bodyLines, "\n")})
			index = next
			if index >= len(lines) {
				break
			}

			title, ok = parseTabTitle(lines[index])
		}

		html, err := r.renderTabs(sections, resolve, options)
		if err != nil {
			return "", err
		}

		out = append(out, "", html, "")
	}

	return strings.Join(out, "\n"), nil
}

// renderTabs renders parsed tab sections while recursively supporting Markdown inside each panel.
func (r *Renderer) renderTabs(sections []tabSection, resolve func(string) string, options Options) (string, error) {
	var output strings.Builder

	output.WriteString(`<div class="markdown-tabs"><div class="markdown-tab-list" role="tablist">`)

	for index, section := range sections {
		class := "markdown-tab"
		selected := "false"

		if index == 0 {
			class += " active"
			selected = "true"
		}

		output.WriteString(`<button type="button" class="` + class + `" role="tab" aria-selected="` + selected + `">`)
		output.WriteString(stdhtml.EscapeString(section.title))
		output.WriteString(`</button>`)
	}

	output.WriteString(`</div><div class="markdown-tab-panels">`)

	for index, section := range sections {
		body, err := r.renderRawResolved(section.body, resolve, options)
		if err != nil {
			return "", err
		}

		class := "markdown-tab-panel"

		if index != 0 {
			class += " markdown-tab-panel-hidden"
		}

		output.WriteString(`<div class="` + class + `" role="tabpanel">`)
		output.WriteString(body)
		output.WriteString(`</div>`)
	}

	output.WriteString(`</div></div>`)
	return output.String(), nil
}

// preprocessDetails converts Material-style collapsible detail blocks into native details elements.
func (r *Renderer) preprocessDetails(source string, resolve func(string) string, options Options) (string, error) {
	lines := strings.Split(source, "\n")
	out := make([]string, 0, len(lines))

	for index := 0; index < len(lines); {
		if marker := fenceDelimiter(lines[index]); marker != "" {
			index = appendFencedBlock(lines, index, marker, &out)
			continue
		}

		title, open, ok := parseDetailsTitle(lines[index])
		if !ok {
			out = append(out, lines[index])

			index++
			continue
		}

		bodyLines, next := indentedBody(lines, index+1)
		body, err := r.renderRawResolved(strings.Join(bodyLines, "\n"), resolve, options)
		if err != nil {
			return "", err
		}

		openAttribute := ""

		if open {
			openAttribute = " open"
		}

		markup := `<details class="markdown-details"` + openAttribute + `><summary>` + stdhtml.EscapeString(
			title,
		) + `</summary><div class="markdown-details-body">` + body + `</div></details>`
		out = append(out, "", markup, "")
		index = next
	}

	return strings.Join(out, "\n"), nil
}

// parseTabTitle parses a top-level tab declaration such as === "Linux".
func parseTabTitle(line string) (title string, ok bool) {
	if strings.TrimLeft(line, " \t") != line {
		return "", false
	}

	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "===") {
		return "", false
	}

	return parseQuotedTitle(strings.TrimSpace(strings.TrimPrefix(trimmed, "===")))
}

// parseDetailsTitle parses ??? and ???+ collapsible block declarations.
func parseDetailsTitle(line string) (title string, open bool, ok bool) {
	if strings.TrimLeft(line, " \t") != line {
		return "", false, false
	}

	trimmed := strings.TrimSpace(line)
	open = strings.HasPrefix(trimmed, "???+")
	prefix := "???"

	if open {
		prefix = "???+"
	} else if !strings.HasPrefix(trimmed, prefix) {
		return "", false, false
	}

	title, ok = parseQuotedTitle(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)))

	return title, open, ok
}

// parseQuotedTitle parses one quoted block title and supports standard Go-style escapes.
func parseQuotedTitle(value string) (title string, ok bool) {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", false
	}

	title, err := strconv.Unquote(value)
	if err != nil || strings.TrimSpace(title) == "" {
		return "", false
	}

	return title, true
}

// indentedBody collects blank and four-space-indented lines following a custom block declaration.
func indentedBody(lines []string, start int) (body []string, next int) {
	body = make([]string, 0)
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

// stripBlockIndent removes one tab or four spaces from a custom block body line.
func stripBlockIndent(line string) (content string, ok bool) {
	if strings.HasPrefix(line, "\t") {
		return strings.TrimPrefix(line, "\t"), true
	}
	if strings.HasPrefix(line, "    ") {
		return strings.TrimPrefix(line, "    "), true
	}

	return "", false
}

// fenceDelimiter returns the Markdown fence marker when a line starts a fenced code block.
func fenceDelimiter(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") {
		return "```"
	}
	if strings.HasPrefix(trimmed, "~~~") {
		return "~~~"
	}

	return ""
}

// appendFencedBlock copies a complete fenced code block without interpreting custom block syntax.
func appendFencedBlock(lines []string, start int, marker string, out *[]string) int {
	*out = append(*out, lines[start])

	for index := start + 1; index < len(lines); index++ {
		*out = append(*out, lines[index])
		if strings.HasPrefix(strings.TrimSpace(lines[index]), marker) {
			return index + 1
		}
	}

	return len(lines)
}

// walkWikiLinks visits wiki links outside fenced code blocks in source order.
func walkWikiLinks(source string, visit func(target, label string)) {
	lines := strings.Split(source, "\n")
	fence := ""

	for _, line := range lines {
		marker := fenceDelimiter(line)
		if fence != "" {
			if marker == fence {
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
func walkWikiLinksLine(line string, visit func(target, label string)) {
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
		target, label, ok := parseWikiLink(line[start+2 : end])

		if ok {
			visit(target, label)
		}

		offset = end + 2
	}
}

// parseWikiLink splits a wiki-link body into target and optional label.
func parseWikiLink(value string) (target string, label string, ok bool) {
	target, label, hasLabel := strings.Cut(value, "|")
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", false
	}

	if !hasLabel || strings.TrimSpace(label) == "" {
		label = target
	} else {
		label = strings.TrimSpace(label)
	}

	return target, label, true
}

// wikiLinkPrefix returns the configured wiki-link URL prefix.
func wikiLinkPrefix(options Options) string {
	if strings.TrimSpace(options.WikiLinkPrefix) == "" {
		return "/pages/"
	}
	return options.WikiLinkPrefix
}

// rewriteWikiLinks converts wiki-link syntax outside fenced code blocks into Markdown links.
func rewriteWikiLinks(source string, resolve func(string) string, prefix string) string {
	lines := strings.Split(source, "\n")
	fence := ""

	for index, line := range lines {
		marker := fenceDelimiter(line)
		if fence != "" {
			if marker == fence {
				fence = ""
			}
			continue
		}
		if marker != "" {
			fence = marker
			continue
		}

		lines[index] = rewriteWikiLinksLine(line, resolve, prefix)
	}

	return strings.Join(lines, "\n")
}

// rewriteWikiLinksLine converts wiki links in one Markdown line without regular expressions.
func rewriteWikiLinksLine(line string, resolve func(string) string, prefix string) string {
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
		target, label, ok := parseWikiLink(line[start+2 : end])
		if !ok {
			output.WriteString(line[offset : end+2])

			offset = end + 2
			continue
		}

		output.WriteString(line[offset:start])
		output.WriteByte('[')
		output.WriteString(label)
		output.WriteString("](")
		output.WriteString(prefix)
		output.WriteString(resolve(target))
		output.WriteByte(')')

		offset = end + 2
	}

	return output.String()
}

// tableDirectivesEnabled reports whether any table directive feature can be rendered.
func tableDirectivesEnabled(options Options) bool {
	return options.Tables && (options.TableStyles || options.TableSorting || options.TableFiltering)
}

// preprocessTableDirectives replaces enabled table directives with trusted markers consumed after rendering.
func preprocessTableDirectives(source string, options Options) string {
	lines := strings.Split(source, "\n")
	out := make([]string, 0, len(lines))
	fence := ""

	for index, line := range lines {
		marker := fenceDelimiter(line)
		if fence != "" {
			out = append(out, line)

			if marker == fence {
				fence = ""
			}
			continue
		}
		if marker != "" {
			fence = marker
			out = append(out, line)
			continue
		}

		trimmed := strings.TrimSpace(line)

		if previousTableLine(lines, index) {
			if directive, ok := parseTableDirective(trimmed); ok && tableDirectiveActive(directive, options) {
				out = append(
					out,
					`<div class="lore-table-style-marker" data-table-style="`+stdhtml.EscapeString(trimmed)+`"></div>`,
				)
				continue
			}
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

// previousTableLine reports whether a directive immediately follows a Markdown table row.
func previousTableLine(lines []string, index int) bool {
	if index == 0 {
		return false
	}

	previous := index - 1

	for previous >= 0 && strings.TrimSpace(lines[previous]) == "" {
		previous--
	}

	return previous >= 0 && strings.Contains(lines[previous], "|")
}

// parseTableDirective parses trusted table colors and optional browser interactions.
func parseTableDirective(line string) (style tableStyle, ok bool) {
	if !strings.HasPrefix(line, "{table ") || !strings.HasSuffix(line, "}") {
		return tableStyle{}, false
	}

	directive := tableStyle{rows: map[int]string{}, columns: map[int]string{}, cells: map[[2]int]string{}}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "{table"), "}"))
	if body == "" {
		return tableStyle{}, false
	}

	for _, token := range strings.Fields(body) {
		switch token {
		case "sortable":
			directive.sortable = true
			continue
		case "filterable":
			directive.filterable = true
			continue
		}

		key, tone, ok := strings.Cut(token, "=")
		if !ok || !tableTone(tone) {
			return tableStyle{}, false
		}
		if key == "header" {
			directive.header = tone
			continue
		}

		kind, target, ok := strings.Cut(key, ":")
		if !ok {
			return tableStyle{}, false
		}

		switch kind {
		case "row":
			row, ok := parsePositiveInt(target)
			if !ok {
				return tableStyle{}, false
			}

			directive.rows[row] = tone
		case "col", "column":
			column, ok := parsePositiveInt(target)
			if !ok {
				return tableStyle{}, false
			}

			directive.columns[column] = tone
		case "cell":
			rowValue, columnValue, ok := strings.Cut(target, ",")
			if !ok {
				return tableStyle{}, false
			}

			row, ok := parsePositiveInt(rowValue)
			if !ok {
				return tableStyle{}, false
			}

			column, ok := parsePositiveInt(columnValue)
			if !ok {
				return tableStyle{}, false
			}

			directive.cells[[2]int{row, column}] = tone
		default:
			return tableStyle{}, false
		}
	}

	return directive, true
}

// parsePositiveInt parses a one-based table row or column index.
func parsePositiveInt(raw string) (value int, ok bool) {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, false
	}

	return value, true
}

// tableDirectiveActive reports whether a parsed directive contains any currently enabled behavior.
func tableDirectiveActive(directive tableStyle, options Options) bool {
	colors := directive.header != "" || len(directive.rows) > 0 || len(directive.columns) > 0 ||
		len(directive.cells) > 0
	return (colors && options.TableStyles) ||
		(directive.sortable && options.TableSorting) ||
		(directive.filterable && options.TableFiltering)
}

// tableTone reports whether a table color maps to a trusted theme-aware palette class.
func tableTone(value string) bool {
	switch value {
	case "accent", "accent-soft", "info", "success", "warning", "danger", "neutral",
		"gray", "blue", "purple", "green", "yellow", "orange", "red":
		return true
	default:
		return false
	}
}

// tableDirectiveWalker tracks the nearest table while applying rendered table markers.
type tableDirectiveWalker struct {
	options   Options
	markers   []*xhtml.Node
	lastTable *xhtml.Node
}

// applyTableDirectiveMarkers applies trusted directives to the nearest preceding rendered table.
func applyTableDirectiveMarkers(rendered string, options Options) (string, error) {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(strings.NewReader(rendered), contextNode)
	if err != nil {
		return "", err
	}

	walker := tableDirectiveWalker{options: options}
	for _, node := range nodes {
		walker.walk(node)
	}

	for _, marker := range walker.markers {
		if marker.Parent != nil {
			marker.Parent.RemoveChild(marker)
		}
	}

	var output strings.Builder

	for _, node := range nodes {
		if err := xhtml.Render(&output, node); err != nil {
			return "", err
		}
	}

	return output.String(), nil
}

// walk applies one table marker in document order and records it for removal.
func (w *tableDirectiveWalker) walk(node *xhtml.Node) {
	if node.Type == xhtml.ElementNode {
		if node.Data == "table" {
			w.lastTable = node
		}

		if node.Data == "div" &&
			strings.Contains(" "+htmlAttribute(node, "class")+" ", " lore-table-style-marker ") {
			if directive, ok := parseTableDirective(htmlAttribute(node, "data-table-style")); ok &&
				w.lastTable != nil {
				applyTableDirective(w.lastTable, directive, w.options)
			}

			w.markers = append(w.markers, node)
			return
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		w.walk(child)
	}
}

// applyTableDirective applies enabled colors and interaction classes to one rendered table.
func applyTableDirective(table *xhtml.Node, directive tableStyle, options Options) {
	if options.TableSorting && directive.sortable {
		addHTMLClass(table, "lore-table-sortable")
	}
	if options.TableFiltering && directive.filterable {
		addHTMLClass(table, "lore-table-filterable")
	}

	if !options.TableStyles {
		return
	}

	rows := tableRows(table)
	if len(rows) == 0 {
		return
	}

	colors := directive.header != "" || len(directive.rows) > 0 || len(directive.columns) > 0 ||
		len(directive.cells) > 0
	if !colors {
		return
	}

	addHTMLClass(table, "lore-table-styled")

	if directive.header != "" {
		for _, cell := range rowCells(rows[0]) {
			setTableTone(cell, directive.header)
		}
	}
	for column, tone := range directive.columns {
		for _, row := range rows {
			cells := rowCells(row)
			if column <= len(cells) {
				setTableTone(cells[column-1], tone)
			}
		}
	}

	bodyRows := rows

	if hasAncestorSection(rows[0], "thead") {
		bodyRows = rows[1:]
	}
	for row, tone := range directive.rows {
		if row > len(bodyRows) {
			continue
		}
		for _, cell := range rowCells(bodyRows[row-1]) {
			setTableTone(cell, tone)
		}
	}
	for position, tone := range directive.cells {
		row, column := position[0], position[1]
		if row > len(bodyRows) {
			continue
		}

		cells := rowCells(bodyRows[row-1])

		if column <= len(cells) {
			setTableTone(cells[column-1], tone)
		}
	}
}

// tableRows returns table rows in document order.
func tableRows(table *xhtml.Node) []*xhtml.Node {
	var rows []*xhtml.Node
	appendTableRows(table, &rows)

	return rows
}

// appendTableRows collects row elements without descending into a row once found.
func appendTableRows(node *xhtml.Node, rows *[]*xhtml.Node) {
	if node.Type == xhtml.ElementNode && node.Data == "tr" {
		*rows = append(*rows, node)
		return
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		appendTableRows(child, rows)
	}
}

// rowCells returns direct th and td children for one rendered row.
func rowCells(row *xhtml.Node) []*xhtml.Node {
	var cells []*xhtml.Node

	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode && (child.Data == "th" || child.Data == "td") {
			cells = append(cells, child)
		}
	}

	return cells
}

// hasAncestorSection reports whether a row sits below the named table section.
func hasAncestorSection(node *xhtml.Node, section string) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Type == xhtml.ElementNode && current.Data == section {
			return true
		}
		if current.Type == xhtml.ElementNode && current.Data == "table" {
			return false
		}
	}
	return false
}

// addHTMLClass adds a class to one rendered HTML element when it is not already present.
func addHTMLClass(node *xhtml.Node, className string) {
	classes := strings.Fields(htmlAttribute(node, "class"))

	for _, existing := range classes {
		if existing == className {
			return
		}
	}

	classes = append(classes, className)

	setHTMLAttribute(node, "class", strings.Join(classes, " "))
}

// setTableTone replaces a previously applied tone with the requested theme-aware class.
func setTableTone(node *xhtml.Node, tone string) {
	const prefix = "table-tone-"
	classes := strings.Fields(htmlAttribute(node, "class"))
	kept := classes[:0]

	for _, className := range classes {
		if !strings.HasPrefix(className, prefix) {
			kept = append(kept, className)
		}
	}

	kept = append(kept, prefix+tone)

	setHTMLAttribute(node, "class", strings.Join(kept, " "))
}

// setHTMLAttribute sets or appends one HTML node attribute.
func setHTMLAttribute(node *xhtml.Node, key, value string) {
	for index := range node.Attr {
		if node.Attr[index].Key == key {
			node.Attr[index].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, xhtml.Attribute{Key: key, Val: value})
}

// htmlHeadingLevel returns the numeric level of an h1-h6 element.
func htmlHeadingLevel(node *xhtml.Node) (level int, ok bool) {
	if node.Type != xhtml.ElementNode || len(node.Data) != 2 {
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
	document, err := xhtml.Parse(strings.NewReader(rendered))
	if err != nil {
		return nil
	}

	var contents []Heading
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if level, ok := htmlHeadingLevel(node); ok {
			id := htmlAttribute(node, "id")
			if id != "" {
				contents = append(contents, Heading{
					Level: level,
					ID:    id,
					Title: strings.TrimSpace(htmlText(node)),
				})
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
func htmlAttribute(node *xhtml.Node, key string) string {
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

// preprocessCallouts converts supported callout syntax into sanitized HTML blocks.
func (r *Renderer) preprocessCallouts(source string, resolve func(string) string, options Options) (string, error) {
	lines := strings.Split(source, "\n")
	out := make([]string, 0, len(lines))

	for index := 0; index < len(lines); index++ {
		if marker := fenceDelimiter(lines[index]); marker != "" {
			next := appendFencedBlock(lines, index, marker, &out)
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

		for index < len(lines) && strings.TrimSpace(lines[index]) != "" {
			body = append(body, strings.TrimSpace(lines[index]))
			index++
		}

		bodyHTML, err := r.renderRawResolved(strings.Join(body, "\n"), resolve, options)
		if err != nil {
			return "", err
		}

		label := strings.ToUpper(kind[:1]) + kind[1:]
		out = append(
			out,
			`<aside class="callout `+kind+`"><strong>`+stdhtml.EscapeString(
				label,
			)+`</strong><div class="callout-body">`+bodyHTML+`</div></aside>`+"\n",
		)
	}

	return strings.Join(out, "\n"), nil
}

// supportedCallout reports whether a callout kind has built-in presentation styling.
func supportedCallout(kind string) bool {
	switch kind {
	case "note", "info", "tip", "success", "warning", "danger", "error":
		return true
	default:
		return false
	}
}
