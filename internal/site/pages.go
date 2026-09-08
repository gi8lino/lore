package site

import (
	"bytes"
	"cmp"
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	md "github.com/gi8lino/lore/internal/markdown"
	"github.com/gi8lino/lore/internal/navigation"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type pageDiscovery struct {
	sourceDir string
	pages     []sourcePage
	routes    map[string]string
}

func discoverPages(sourceDir string) ([]sourcePage, error) {
	discovery := pageDiscovery{
		sourceDir: sourceDir,
		routes:    make(map[string]string),
	}

	if err := filepath.WalkDir(sourceDir, discovery.visit); err != nil {
		return nil, err
	}

	slices.SortFunc(discovery.pages, compareSourcePages)
	return discovery.pages, nil
}

func (d *pageDiscovery) visit(filename string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if filename == d.sourceDir {
		return nil
	}
	if entry.IsDir() {
		if strings.HasPrefix(entry.Name(), ".") {
			return filepath.SkipDir
		}
		return nil
	}
	if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
		return nil
	}

	relative, err := filepath.Rel(d.sourceDir, filename)
	if err != nil {
		return err
	}
	relative = filepath.ToSlash(relative)

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	route := markdownFileRoute(relative)
	if existing, found := d.routes[route]; found {
		return fmt.Errorf("markdown files %s and %s map to the same route %q", existing, relative, route)
	}

	d.routes[route] = relative
	title, hasTitle := markdownTitle(string(data), route)
	d.pages = append(d.pages, sourcePage{
		SourcePath:      relative,
		Route:           route,
		Title:           title,
		Markdown:        string(data),
		HasTitleHeading: hasTitle,
	})

	return nil
}

func compareSourcePages(left, right sourcePage) int {
	return cmp.Compare(left.SourcePath, right.SourcePath)
}

func hasHomePage(pages []sourcePage) bool {
	for _, page := range pages {
		if page.Route == "" {
			return true
		}
	}

	return false
}

func indexPages(pages []sourcePage) (map[string]string, map[string]string) {
	routesBySource := make(map[string]string, len(pages))
	wikiTargets := make(map[string]string, len(pages)*2)
	ambiguousWikiTargets := make(map[string]bool)

	for _, page := range pages {
		routesBySource[page.SourcePath] = page.Route
		registerWikiTarget(wikiTargets, ambiguousWikiTargets, md.Slug(page.Route), page.Route)
		registerWikiTarget(wikiTargets, ambiguousWikiTargets, md.Slug(page.Title), page.Route)
	}
	for target := range ambiguousWikiTargets {
		delete(wikiTargets, target)
	}

	return routesBySource, wikiTargets
}

func markdownFileRoute(filename string) string {
	clean := strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(filename)), "/")
	clean = strings.TrimSuffix(clean, path.Ext(clean))

	if path.Base(clean) == "index" {
		clean = path.Dir(clean)
		if clean == "." {
			clean = ""
		}
	}

	return strings.Trim(clean, "/")
}

func markdownTitle(source, route string) (title string, hasTitle bool) {
	lines := strings.Split(strings.TrimPrefix(source, "\ufeff"), "\n")
	fence := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if fence == "" {
				fence = marker
			} else if marker == fence {
				fence = ""
			}
			continue
		}

		if fence == "" && strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if title != "" {
				return title, true
			}
		}
	}

	if route == "" {
		return "Home", false
	}

	segment := path.Base(route)
	segment = strings.NewReplacer("-", " ", "_", " ").Replace(segment)
	words := strings.Fields(segment)
	for index, word := range words {
		runes := []rune(word)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[index] = string(runes)
		}
	}

	return strings.Join(words, " "), false
}

func registerWikiTarget(targets map[string]string, ambiguous map[string]bool, target, route string) {
	target = strings.Trim(target, "/")
	if target == "" && route != "" {
		return
	}
	if existing, found := targets[target]; found && existing != route {
		ambiguous[target] = true
		return
	}

	targets[target] = route
}

func validateWikiLinks(page sourcePage, targets map[string]string) error {
	for _, target := range md.Links(page.Markdown) {
		if _, found := targets[target]; !found {
			return fmt.Errorf("%s contains unresolved wiki link %q", page.SourcePath, target)
		}
	}

	return nil
}

func expandedPrefixes(route string) []string {
	parts := strings.Split(strings.Trim(route, "/"), "/")
	if len(parts) <= 1 {
		return nil
	}

	expanded := make([]string, 0, len(parts)-1)
	for index := 1; index < len(parts); index++ {
		expanded = append(expanded, strings.Join(parts[:index], "/"))
	}

	return expanded
}

func subpagesRenderer(tree []navigation.Node, route, basePath string) md.SubpagesRenderer {
	children := tree
	if strings.Trim(route, "/") != "" {
		children = navigation.Children(tree, route)
	}

	return func(options md.SubpagesOptions) (string, error) {
		if len(children) == 0 {
			return "", nil
		}

		label := options.Title
		if !options.ShowTitle || strings.TrimSpace(label) == "" {
			label = "Pages in this section"
		}

		var output strings.Builder
		output.WriteString(`<nav class="subpage-toc" aria-label="`)
		output.WriteString(template.HTMLEscapeString(label))
		output.WriteString(`">`)
		if options.ShowTitle {
			output.WriteString(`<div class="subpage-toc-heading"><h2>`)
			output.WriteString(template.HTMLEscapeString(options.Title))
			output.WriteString(`</h2></div>`)
		}
		output.WriteString(`<ul class="subpage-toc-list subpage-toc-root">`)
		for _, child := range children {
			renderSubpageNode(&output, child, basePath)
		}
		output.WriteString(`</ul></nav>`)

		return output.String(), nil
	}
}

func renderSubpageNode(output *strings.Builder, node navigation.Node, basePath string) {
	output.WriteString(`<li class="subpage-toc-item">`)
	if node.Page {
		output.WriteString(`<a class="subpage-toc-link" href="`)
		output.WriteString(template.HTMLEscapeString(pageURL(basePath, node.Slug)))
		output.WriteString(`"><span>`)
		output.WriteString(template.HTMLEscapeString(node.Title))
		output.WriteString(`</span></a>`)
	} else {
		output.WriteString(`<span class="subpage-toc-label"><span>`)
		output.WriteString(template.HTMLEscapeString(node.Title))
		output.WriteString(`</span></span>`)
	}

	if len(node.Children) > 0 {
		output.WriteString(`<ul class="subpage-toc-list">`)
		for _, child := range node.Children {
			renderSubpageNode(output, child, basePath)
		}
		output.WriteString(`</ul>`)
	}

	output.WriteString(`</li>`)
}

func processRenderedHTML(
	rendered, sourcePath string,
	removeTitle bool,
	routesBySource map[string]string,
	basePath string,
) (renderedHTML string, searchText string, err error) {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(strings.NewReader(rendered), contextNode)
	if err != nil {
		return "", "", err
	}

	if removeTitle {
		nodes = removeFirstHeading(nodes)
	}
	for _, node := range nodes {
		if err := rewriteHTMLURLs(node, sourcePath, routesBySource, basePath); err != nil {
			return "", "", err
		}
	}

	var htmlOutput bytes.Buffer
	for _, node := range nodes {
		if err := xhtml.Render(&htmlOutput, node); err != nil {
			return "", "", err
		}
	}

	return htmlOutput.String(), normalizeSearchText(textFromNodes(nodes)), nil
}

func removeFirstHeading(nodes []*xhtml.Node) []*xhtml.Node {
	for index, node := range nodes {
		if node.Type == xhtml.ElementNode && node.Data == "h1" {
			return append(nodes[:index], nodes[index+1:]...)
		}
	}

	return nodes
}

// isRewritableURLAttribute reports whether a static-page attribute contains a navigable local URL.
func isRewritableURLAttribute(element, attribute string) bool {
	switch element {
	case "a":
		return attribute == "href"
	case "img":
		return attribute == "src"
	default:
		return false
	}
}

func rewriteHTMLURLs(node *xhtml.Node, sourcePath string, routesBySource map[string]string, basePath string) error {
	if node.Type == xhtml.ElementNode {
		for index := range node.Attr {
			attribute := &node.Attr[index]
			if !isRewritableURLAttribute(node.Data, attribute.Key) {
				continue
			}

			rewritten, err := rewriteLocalURL(attribute.Val, sourcePath, routesBySource, basePath)
			if err != nil {
				return err
			}
			attribute.Val = rewritten
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if err := rewriteHTMLURLs(child, sourcePath, routesBySource, basePath); err != nil {
			return err
		}
	}

	return nil
}

// isRewritableLocalURL reports whether a parsed URL refers to a non-empty path inside the generated site.
func isRewritableLocalURL(value string, parsed *url.URL) bool {
	if parsed.IsAbs() || parsed.Host != "" {
		return false
	}
	if strings.HasPrefix(value, "//") {
		return false
	}

	return parsed.Path != ""
}

func rewriteLocalURL(value, sourcePath string, routesBySource map[string]string, basePath string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "#") {
		return value, nil
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if !isRewritableLocalURL(value, parsed) {
		return value, nil
	}

	basePath = ensureBasePath(basePath)
	if basePath != "/" && strings.HasPrefix(parsed.Path, basePath) {
		return value, nil
	}

	trailingSlash := strings.HasSuffix(parsed.Path, "/")
	resolved := resolveLocalPath(parsed.Path, sourcePath)
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", fmt.Errorf("link %q escapes the documentation source", value)
	}

	if strings.EqualFold(path.Ext(resolved), ".md") {
		route, found := routesBySource[resolved]
		if !found {
			return "", fmt.Errorf("markdown link %q points to missing file %s", value, resolved)
		}
		parsed.Path = pageURL(basePath, route)
	} else {
		parsed.Path = basePath + strings.TrimPrefix(resolved, "/")
		if trailingSlash && !strings.HasSuffix(parsed.Path, "/") {
			parsed.Path += "/"
		}
	}

	return parsed.String(), nil
}

func resolveLocalPath(value, sourcePath string) string {
	if strings.HasPrefix(value, "/") {
		return strings.TrimPrefix(path.Clean(value), "/")
	}

	resolved := path.Clean(path.Join(path.Dir(sourcePath), value))
	if resolved == "." {
		return ""
	}

	return resolved
}

func textFromNodes(nodes []*xhtml.Node) string {
	var output strings.Builder
	for _, node := range nodes {
		appendNodeText(&output, node)
	}

	return output.String()
}

func appendNodeText(output *strings.Builder, node *xhtml.Node) {
	if node.Type == xhtml.TextNode {
		output.WriteString(node.Data)
		output.WriteByte(' ')
	}
	if node.Type == xhtml.ElementNode && (node.Data == "script" || node.Data == "style") {
		return
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		appendNodeText(output, child)
	}
}

func normalizeSearchText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
