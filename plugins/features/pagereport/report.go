// Package pagereport parses and renders dynamic page-query reports.
package pagereport

import (
	"cmp"
	"context"
	"fmt"
	"html/template"
	"slices"
	"strconv"
	"strings"

	"github.com/gi8lino/lore/pluginapi"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Options controls one {{pages}} report invocation.
type Options struct {
	Query   string
	Columns []string
	View    string
	Sort    string
	Limit   int
}

// Source supplies page discovery and full page metadata for a report.
type Source interface {
	Search(context.Context, string, int) ([]pluginapi.Page, error)
	GetPage(context.Context, string) (pluginapi.Page, error)
}

// Parse recognizes one standalone {{pages ...}} invocation.
func Parse(line string) (Options, bool) {
	body, ok := pageReportBody(line)
	if !ok {
		return Options{}, false
	}

	arguments, ok := parseArguments(body)
	if !ok {
		return Options{}, false
	}

	return optionsFromArguments(arguments)
}

// pageReportBody extracts the option text from a standalone {{pages ...}} invocation.
func pageReportBody(line string) (string, bool) {
	value := strings.TrimSpace(line)
	body, ok := strings.CutPrefix(value, "{{pages")
	if !ok {
		return "", false
	}

	body, ok = strings.CutSuffix(body, "}}")
	if !ok || (body != "" && body[0] != ' ' && body[0] != '\t') {
		return "", false
	}

	return strings.TrimSpace(body), true
}

// optionsFromArguments normalizes parsed arguments and validates report options.
func optionsFromArguments(arguments map[string]string) (Options, bool) {
	options := Options{
		Query:   strings.TrimSpace(arguments["query"]),
		Columns: []string{"title", "status", "owner", "updated"},
		View:    cmp.Or(strings.TrimSpace(arguments["view"]), "table"),
		Sort:    cmp.Or(strings.TrimSpace(arguments["sort"]), "relevance"),
		Limit:   defaultLimit,
	}

	if options.Query == "" {
		return Options{}, false
	}

	if value := strings.TrimSpace(arguments["columns"]); value != "" {
		options.Columns = splitColumns(value)
	}
	if len(options.Columns) == 0 {
		return Options{}, false
	}

	limit, ok := reportLimit(arguments["limit"])
	if !ok {
		return Options{}, false
	}
	options.Limit = limit

	if !validOptions(options) {
		return Options{}, false
	}

	return options, true
}

// reportLimit parses an optional bounded report limit.
func reportLimit(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultLimit, true
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > maxLimit {
		return 0, false
	}

	return limit, true
}

// validOptions reports whether the report presentation and columns are supported.
func validOptions(options Options) bool {
	if options.View != "table" && options.View != "list" && options.View != "cards" {
		return false
	}
	if options.Sort != "relevance" && options.Sort != "updated" && options.Sort != "title" && options.Sort != "path" {
		return false
	}

	for _, column := range options.Columns {
		if !validColumn(column) {
			return false
		}
	}

	return true
}

// NewRenderer returns a request-bound renderer backed by the page catalog.
func NewRenderer(ctx context.Context, source Source) func(Options) (string, error) {
	return func(options Options) (string, error) {
		pages, err := source.Search(ctx, options.Query, options.Limit)
		if err != nil {
			return "", err
		}

		for index := range pages {
			page, err := source.GetPage(ctx, pages[index].Slug)
			if err != nil {
				return "", err
			}
			pages[index] = page
		}

		sortPages(pages, options.Sort)
		return render(options, pages)
	}
}

type argumentParser struct {
	value string
	index int
}

// parseArguments parses key=value options without using regular expressions.
func parseArguments(value string) (map[string]string, bool) {
	result := map[string]string{}
	parser := argumentParser{value: value}

	for {
		parser.skipSpace()
		if parser.index == len(parser.value) {
			return result, true
		}

		name := parser.readName()
		if name == "" {
			return nil, false
		}
		parser.skipSpace()
		if parser.index >= len(parser.value) || parser.value[parser.index] != '=' {
			return nil, false
		}
		parser.index++
		parser.skipSpace()

		value, ok := parser.readValue()
		if !ok {
			return nil, false
		}
		if _, exists := result[name]; exists {
			return nil, false
		}
		result[name] = value
	}
}

// skipSpace advances past spaces and horizontal tabs.
func (p *argumentParser) skipSpace() {
	for p.index < len(p.value) && (p.value[p.index] == ' ' || p.value[p.index] == '\t') {
		p.index++
	}
}

// readName consumes one lowercase option name.
func (p *argumentParser) readName() string {
	start := p.index
	for p.index < len(p.value) {
		character := p.value[p.index]
		if (character >= 'a' && character <= 'z') || character == '_' {
			p.index++
			continue
		}
		break
	}
	return p.value[start:p.index]
}

// readValue consumes one quoted or unquoted option value.
func (p *argumentParser) readValue() (string, bool) {
	if p.index >= len(p.value) {
		return "", false
	}
	if p.value[p.index] != '"' {
		return p.readBareValue()
	}

	return p.readQuotedValue()
}

// readBareValue consumes a value up to the next horizontal whitespace.
func (p *argumentParser) readBareValue() (string, bool) {
	start := p.index

	for p.index < len(p.value) && p.value[p.index] != ' ' && p.value[p.index] != '\t' {
		p.index++
	}

	return p.value[start:p.index], p.index > start
}

// readQuotedValue consumes and unquotes a double-quoted option value.
func (p *argumentParser) readQuotedValue() (string, bool) {
	start := p.index
	p.index++
	escaped := false

	for p.index < len(p.value) {
		character := p.value[p.index]
		p.index++

		if escaped {
			escaped = false
			continue
		}
		if character == '\\' {
			escaped = true
			continue
		}
		if character != '"' {
			continue
		}

		decoded, err := strconv.Unquote(p.value[start:p.index])
		return decoded, err == nil
	}

	return "", false
}

// splitColumns normalizes a comma-separated report column list.
func splitColumns(value string) []string {
	columns := make([]string, 0)
	for column := range strings.SplitSeq(value, ",") {
		column = strings.TrimSpace(column)
		if column != "" {
			columns = append(columns, column)
		}
	}
	return columns
}

// validColumn reports whether a report column is supported.
func validColumn(column string) bool {
	switch column {
	case "title", "path", "status", "owner", "updated", "author", "tags", "views":
		return true
	}
	key, ok := strings.CutPrefix(column, "property:")
	return ok && strings.TrimSpace(key) != ""
}

// sortPages applies the configured deterministic report ordering.
func sortPages(pages []pluginapi.Page, sort string) {
	switch sort {
	case "title":
		slices.SortFunc(pages, func(left, right pluginapi.Page) int {
			return strings.Compare(strings.ToLower(left.Title), strings.ToLower(right.Title))
		})
	case "path":
		slices.SortFunc(pages, func(left, right pluginapi.Page) int {
			return strings.Compare(strings.ToLower(left.Slug), strings.ToLower(right.Slug))
		})
	case "updated":
		slices.SortFunc(pages, func(left, right pluginapi.Page) int {
			return right.UpdatedAt.Compare(left.UpdatedAt)
		})
	}
}

type tableData struct {
	Columns []columnData
	Rows    []rowData
}

type columnData struct {
	Key   string
	Label string
}

type rowData struct {
	Slug  string
	Cells []string
}

var reportTemplate = template.Must(template.New("page-report").Parse(`{{ define "table" }}<div class="lore-page-report lore-table-scroll"><table class="lore-page-report-table"><thead><tr>{{ range .Columns }}<th>{{ .Label }}</th>{{ end }}</tr></thead><tbody>{{ range .Rows }}<tr>{{ $slug := .Slug }}{{ range $index, $cell := .Cells }}<td>{{ if eq $index 0 }}<a href="/pages/{{ $slug }}">{{ $cell }}</a>{{ else }}{{ $cell }}{{ end }}</td>{{ end }}</tr>{{ else }}<tr><td colspan="{{ len .Columns }}"><span class="muted">No pages match this query.</span></td></tr>{{ end }}</tbody></table></div>{{ end }}
{{ define "list" }}<div class="lore-page-report"><ul class="lore-page-report-list">{{ range .Rows }}<li><a href="/pages/{{ .Slug }}">{{ index .Cells 0 }}</a>{{ range $index, $cell := .Cells }}{{ if gt $index 0 }}<span>{{ $cell }}</span>{{ end }}{{ end }}</li>{{ else }}<li class="muted">No pages match this query.</li>{{ end }}</ul></div>{{ end }}
{{ define "cards" }}<div class="lore-page-report lore-page-report-cards">{{ range .Rows }}<a class="lore-page-report-card" href="/pages/{{ .Slug }}"><strong>{{ index .Cells 0 }}</strong>{{ range $index, $cell := .Cells }}{{ if gt $index 0 }}<span>{{ $cell }}</span>{{ end }}{{ end }}</a>{{ else }}<p class="muted">No pages match this query.</p>{{ end }}</div>{{ end }}`))

// render executes the selected report presentation for the resolved pages.
func render(options Options, pages []pluginapi.Page) (string, error) {
	data := tableData{Columns: make([]columnData, 0, len(options.Columns)), Rows: make([]rowData, 0, len(pages))}
	for _, column := range options.Columns {
		data.Columns = append(data.Columns, columnData{Key: column, Label: columnLabel(column)})
	}
	for _, page := range pages {
		cells := make([]string, 0, len(options.Columns))
		for _, column := range options.Columns {
			cells = append(cells, pageValue(page, column))
		}
		data.Rows = append(data.Rows, rowData{Slug: page.Slug, Cells: cells})
	}

	var output strings.Builder
	if err := reportTemplate.ExecuteTemplate(&output, options.View, data); err != nil {
		return "", fmt.Errorf("render page report: %w", err)
	}
	return output.String(), nil
}

// columnLabel returns the human-readable heading for a report column.
func columnLabel(column string) string {
	if key, ok := strings.CutPrefix(column, "property:"); ok {
		return key
	}
	return strings.ToUpper(column[:1]) + column[1:]
}

// pageValue resolves one report cell from page metadata.
func pageValue(page pluginapi.Page, column string) string {
	switch column {
	case "title":
		return page.Title
	case "path":
		return page.Slug
	case "status":
		return page.Status
	case "owner":
		return page.OwnerGroup
	case "updated":
		return page.UpdatedAt.Format("2006-01-02")
	case "author":
		return page.Author
	case "tags":
		return strings.Join(page.Tags, ", ")
	case "views":
		return strconv.FormatInt(page.ViewCount, 10)
	}
	if key, ok := strings.CutPrefix(column, "property:"); ok {
		for _, property := range page.Properties {
			if strings.EqualFold(property.Key, key) {
				return property.Value
			}
		}
	}
	return ""
}
