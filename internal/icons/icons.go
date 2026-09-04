// Package icons exposes the Lucide icons used by the wiki interface.
package icons

import (
	"html/template"
	"slices"
	"strings"

	lucide "github.com/kaugesaar/lucide-go"
)

//go:generate go run ../../scripts/generate-icons

// Option describes one Lucide icon available to wiki icon pickers.
type Option struct {
	// Name is the persisted Lucide icon identifier.
	Name string
	// Label is the human-readable name shown in the icon picker.
	Label string
}

// SVG renders a Lucide icon at the requested pixel size.
func SVG(name string, size int) template.HTML {
	if size <= 0 {
		size = 20
	}
	return lucide.Icon(strings.TrimSpace(name), map[string]any{
		"size":  size,
		"class": "lucide-icon",
	})
}

// NavigationOptions returns every icon generated from the installed Lucide catalog.
func NavigationOptions() []Option {
	return slices.Clone(navigationOptions)
}

// Search returns at most limit generated icons matching a name or label.
func Search(query string, limit int) []Option {
	options, _ := SearchPage(query, 0, limit)
	return options
}

// SearchPage returns one page of generated icons and whether another page is available.
func SearchPage(query string, offset, limit int) ([]Option, bool) {
	if limit <= 0 {
		return nil, false
	}
	if offset < 0 {
		offset = 0
	}
	query = strings.ToLower(strings.TrimSpace(query))
	result := make([]Option, 0, min(limit+1, len(navigationOptions)))
	matched := 0
	for _, option := range navigationOptions {
		if query != "" && !strings.Contains(option.Name, query) &&
			!strings.Contains(strings.ToLower(option.Label), query) {
			continue
		}
		if matched < offset {
			matched++
			continue
		}
		result = append(result, option)
		if len(result) > limit {
			break
		}
	}
	if len(result) > limit {
		return result[:limit], true
	}
	return result, false
}

// IsNavigationIcon reports whether a persisted page or navigation icon is allowed.
func IsNavigationIcon(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}
	_, found := slices.BinarySearchFunc(navigationOptions, name, func(option Option, candidate string) int {
		return strings.Compare(option.Name, candidate)
	})
	return found
}
