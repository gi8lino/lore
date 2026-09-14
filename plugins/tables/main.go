package main

import (
	"github.com/gi8lino/lore/pluginapi"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"strings"
)

func main() {}
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	enabled := func(name string) bool { v, ok := request.Features["io.lore.tables."+name]; return !ok || v }
	options := Options{Tables: enabled("tables"), TableStyles: enabled("styles"), TableSorting: enabled("sorting"), TableFiltering: enabled("filtering")}
	output := request.Source
	switch request.Stage {
	case "preprocess":
		if tableDirectivesEnabled(options) {
			output = preprocessTableDirectives(output, options)
		}
	case "postprocess":
		var err error
		if tableDirectivesEnabled(options) {
			output, err = applyTableDirectiveMarkers(output, options)
			if err != nil {
				return pluginapi.RenderResult{Error: err.Error()}
			}
		}
		output, err = markTables(output)
		if err != nil {
			return pluginapi.RenderResult{Error: err.Error()}
		}
	default:
		return pluginapi.RenderResult{Error: "unsupported stage"}
	}
	return pluginapi.RenderResult{Parts: []pluginapi.RenderPart{{Text: output}}}
}

// markTables leaves semantic HTML as the accessible, script-free fallback.
func markTables(source string) (string, error) {
	nodes, err := xhtml.ParseFragment(strings.NewReader(source), &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return "", err
	}
	root := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, n := range nodes {
		root.AppendChild(n)
	}
	var tables []*xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && n.Data == "table" {
			tables = append(tables, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	for _, table := range tables {
		block := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div, Attr: []xhtml.Attribute{{Key: "class", Val: "lore-plugin-block"}, {Key: "data-lore-plugin", Val: "io.lore.tables"}, {Key: "data-lore-input", Val: "html"}}}
		classes := " " + htmlAttribute(table, "class") + " "
		if strings.Contains(classes, " lore-table-sortable ") || strings.Contains(classes, " lore-table-filterable ") {
			block.Attr = append(block.Attr, xhtml.Attribute{Key: "data-lore-module", Val: "interactive"})
		}
		table.Parent.InsertBefore(block, table)
		table.Parent.RemoveChild(table)
		fallback := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div, Attr: []xhtml.Attribute{{Key: "data-lore-fallback", Val: ""}}}
		block.AppendChild(fallback)
		fallback.AppendChild(table)
	}
	var out strings.Builder
	for n := root.FirstChild; n != nil; n = n.NextSibling {
		if err := xhtml.Render(&out, n); err != nil {
			return "", err
		}
	}
	return out.String(), nil
}
