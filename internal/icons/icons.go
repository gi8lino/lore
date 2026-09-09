// Package icons exposes the icon catalogs used by the Lore interface.
package icons

import (
	"cmp"
	"fmt"
	"html"
	"html/template"
	"slices"
	"strings"

	simpleicons "github.com/go-icons/simple-icons"
	lucide "github.com/kaugesaar/lucide-go"
)

const (
	lucideSuffix = "-lucide"
	simpleSuffix = "-simple"
)

//go:generate go run ../../scripts/generate-icons

// Option describes one icon available to icon pickers.
type Option struct {
	// Name is the persisted icon identifier including its source suffix.
	Name string
	// Label is the human-readable name shown in the icon picker.
	Label string
	// Source is the icon pack shown as secondary picker metadata.
	Source string
}

var iconOptions = buildOptions()

// SVG renders an icon at the requested pixel size.
func SVG(name string, size int) template.HTML {
	name = strings.TrimSpace(name)
	if size <= 0 {
		size = 20
	}

	if lucideName, ok := strings.CutSuffix(name, lucideSuffix); ok {
		return lucide.Icon(lucideName, map[string]any{
			"size":  size,
			"class": "lucide-icon",
		})
	}

	if simpleName, ok := strings.CutSuffix(name, simpleSuffix); ok {
		return simpleSVG(simpleName, size)
	}

	return ""
}

// Options returns every icon from the installed Lucide and Simple Icons catalogs.
func Options() []Option {
	return slices.Clone(iconOptions)
}

// Search returns at most limit icons matching a name or label.
func Search(query string, limit int) []Option {
	options, _ := SearchPage(query, 0, limit)
	return options
}

// SearchPage returns one page of icons and whether another page is available.
func SearchPage(query string, offset, limit int) (options []Option, hasMore bool) {
	if limit <= 0 {
		return nil, false
	}

	offset = max(offset, 0)
	query = strings.ToLower(strings.TrimSpace(query))
	options = make([]Option, 0, min(limit+1, len(iconOptions)))
	matched := 0

	for _, option := range iconOptions {
		if !matches(option, query) {
			continue
		}
		if matched < offset {
			matched++
			continue
		}

		options = append(options, option)
		if len(options) > limit {
			break
		}
	}

	if len(options) > limit {
		return options[:limit], true
	}

	return options, false
}

// IsIcon reports whether a persisted icon identifier is available.
func IsIcon(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}

	_, found := slices.BinarySearchFunc(iconOptions, name, func(option Option, candidate string) int {
		return cmp.Compare(option.Name, candidate)
	})

	return found
}

// buildOptions combines both icon catalogs into one deterministic searchable list.
func buildOptions() []Option {
	options := make([]Option, 0, len(lucideOptions)+len(simpleicons.Names()))
	options = append(options, lucideOptions...)

	for _, name := range simpleicons.Names() {
		options = append(options, Option{
			Name:   name + simpleSuffix,
			Label:  simpleLabel(name),
			Source: "Simple Icons",
		})
	}

	slices.SortFunc(options, func(left, right Option) int {
		return cmp.Compare(left.Name, right.Name)
	})

	return options
}

// matches reports whether an icon name or label contains the normalized query.
func matches(option Option, query string) bool {
	if query == "" {
		return true
	}
	if strings.Contains(option.Name, query) {
		return true
	}

	return strings.Contains(strings.ToLower(option.Label), query)
}

// simpleLabel returns the official title embedded in a Simple Icons SVG.
func simpleLabel(name string) string {
	svg := simpleicons.Icon(name)
	start := strings.Index(svg, "<title>")
	if start < 0 {
		return name
	}

	start += len("<title>")
	end := strings.Index(svg[start:], "</title>")
	if end < 0 {
		return name
	}

	return html.UnescapeString(svg[start : start+end])
}

// simpleSVG adapts an embedded Simple Icons document to Lore's shared icon styling.
func simpleSVG(name string, size int) template.HTML {
	svg := simpleicons.Icon(name)
	if svg == "" {
		return ""
	}

	attributes := fmt.Sprintf(
		`<svg class="simple-icon" width="%d" height="%d" fill="currentColor" aria-hidden="true" focusable="false" `,
		size,
		size,
	)
	svg = strings.Replace(svg, "<svg ", attributes, 1)

	return template.HTML(svg)
}
