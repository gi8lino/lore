package main

import (
	"github.com/gi8lino/lore/pluginapi/blocksyntax"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	stdhtml "html"
	"slices"
	"strconv"
	"strings"
)

type Options struct{ Tables, TableStyles, TableSorting, TableFiltering bool }

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

// tableDirectivesEnabled reports whether any table directive feature can be rendered.
func tableDirectivesEnabled(options Options) bool {
	return options.Tables &&
		(options.TableStyles ||
			options.TableSorting ||
			options.TableFiltering)
}

// preprocessTableDirectives replaces enabled table directives with trusted markers consumed after rendering.
func preprocessTableDirectives(
	source string,
	options Options,
) string {
	lines := strings.Split(source, "\n")
	out := make([]string, 0, len(lines))
	fence := ""

	for index, line := range lines {
		marker := blocksyntax.Fence(line)

		if fence != "" {
			out = append(out, line)

			if blocksyntax.Closes(line, fence) {
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
			directive, ok := parseTableDirective(trimmed)

			if ok &&
				tableDirectiveActive(directive, options) {
				out = append(
					out,
					`<div class="lore-table-style-marker" data-table-style="`+
						stdhtml.EscapeString(trimmed)+
						`"></div>`,
				)

				continue
			}
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

// previousTableLine reports whether a directive immediately follows a Markdown table row.
func previousTableLine(
	lines []string,
	index int,
) bool {
	for _, line := range slices.Backward(lines[:index]) {
		if strings.TrimSpace(line) == "" {
			continue
		}

		return strings.Contains(line, "|")
	}

	return false
}

// parseTableDirective parses trusted table colors and optional browser interactions.
func parseTableDirective(
	line string,
) (style tableStyle, ok bool) {
	body, ok := strings.CutPrefix(line, "{table ")
	if !ok {
		return tableStyle{}, false
	}

	body, ok = strings.CutSuffix(body, "}")
	if !ok {
		return tableStyle{}, false
	}

	directive := tableStyle{
		rows:    map[int]string{},
		columns: map[int]string{},
		cells:   map[[2]int]string{},
	}

	body = strings.TrimSpace(body)

	if body == "" {
		return tableStyle{}, false
	}

	for token := range strings.FieldsSeq(body) {
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
func parsePositiveInt(
	raw string,
) (value int, ok bool) {
	value, err := strconv.Atoi(raw)

	if err != nil || value < 1 {
		return 0, false
	}

	return value, true
}

// tableDirectiveActive reports whether a parsed directive contains any currently enabled behavior.
func tableDirectiveActive(
	directive tableStyle,
	options Options,
) bool {
	colors :=
		directive.header != "" ||
			len(directive.rows) > 0 ||
			len(directive.columns) > 0 ||
			len(directive.cells) > 0

	return (colors && options.TableStyles) ||
		(directive.sortable && options.TableSorting) ||
		(directive.filterable && options.TableFiltering)
}

// tableTone reports whether a table color maps to a trusted theme-aware palette class.
func tableTone(value string) bool {
	switch value {
	case "accent",
		"accent-soft",
		"info",
		"success",
		"warning",
		"danger",
		"neutral",
		"gray",
		"blue",
		"purple",
		"green",
		"yellow",
		"orange",
		"red":
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
func applyTableDirectiveMarkers(
	rendered string,
	options Options,
) (string, error) {
	contextNode := &xhtml.Node{
		Type:     xhtml.ElementNode,
		DataAtom: atom.Div,
		Data:     "div",
	}

	nodes, err := xhtml.ParseFragment(
		strings.NewReader(rendered),
		contextNode,
	)
	if err != nil {
		return "", err
	}

	walker := tableDirectiveWalker{
		options: options,
	}

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
func (w *tableDirectiveWalker) walk(
	node *xhtml.Node,
) {
	if node.Type == xhtml.ElementNode {
		if node.Data == "table" {
			w.lastTable = node
		}

		if node.Data == "div" &&
			strings.Contains(
				" "+htmlAttribute(node, "class")+" ",
				" lore-table-style-marker ",
			) {
			directive, ok := parseTableDirective(
				htmlAttribute(node, "data-table-style"),
			)

			if ok && w.lastTable != nil {
				applyTableDirective(
					w.lastTable,
					directive,
					w.options,
				)
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
func applyTableDirective(
	table *xhtml.Node,
	directive tableStyle,
	options Options,
) {
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

	colors :=
		directive.header != "" ||
			len(directive.rows) > 0 ||
			len(directive.columns) > 0 ||
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
func tableRows(
	table *xhtml.Node,
) []*xhtml.Node {
	var rows []*xhtml.Node

	appendTableRows(table, &rows)

	return rows
}

// appendTableRows collects row elements without descending into a row once found.
func appendTableRows(
	node *xhtml.Node,
	rows *[]*xhtml.Node,
) {
	if node.Type == xhtml.ElementNode &&
		node.Data == "tr" {
		*rows = append(*rows, node)
		return
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		appendTableRows(child, rows)
	}
}

// rowCells returns direct th and td children for one rendered row.
func rowCells(
	row *xhtml.Node,
) []*xhtml.Node {
	var cells []*xhtml.Node

	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode &&
			(child.Data == "th" || child.Data == "td") {
			cells = append(cells, child)
		}
	}

	return cells
}

// hasAncestorSection reports whether a row sits below the named table section.
func hasAncestorSection(
	node *xhtml.Node,
	section string,
) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Type == xhtml.ElementNode &&
			current.Data == section {
			return true
		}

		if current.Type == xhtml.ElementNode &&
			current.Data == "table" {
			return false
		}
	}

	return false
}

// addHTMLClass adds a class to one rendered HTML element when it is not already present.
func addHTMLClass(
	node *xhtml.Node,
	className string,
) {
	classes := strings.Fields(
		htmlAttribute(node, "class"),
	)

	if slices.Contains(classes, className) {
		return
	}

	classes = append(classes, className)

	setHTMLAttribute(
		node,
		"class",
		strings.Join(classes, " "),
	)
}

// setTableTone replaces a previously applied tone with the requested theme-aware class.
func setTableTone(
	node *xhtml.Node,
	tone string,
) {
	const prefix = "table-tone-"

	classes := strings.Fields(
		htmlAttribute(node, "class"),
	)

	classes = slices.DeleteFunc(
		classes,
		func(className string) bool {
			return strings.HasPrefix(
				className,
				prefix,
			)
		},
	)

	classes = append(classes, prefix+tone)

	setHTMLAttribute(
		node,
		"class",
		strings.Join(classes, " "),
	)
}

// setHTMLAttribute sets or appends one HTML node attribute.
func setHTMLAttribute(
	node *xhtml.Node,
	key,
	value string,
) {
	for index := range node.Attr {
		if node.Attr[index].Key == key {
			node.Attr[index].Val = value
			return
		}
	}

	node.Attr = append(
		node.Attr,
		xhtml.Attribute{
			Key: key,
			Val: value,
		},
	)
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
