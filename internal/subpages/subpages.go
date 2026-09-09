// Package subpages parses and renders the configurable {{subpages}} page function.
package subpages

import (
	"strconv"
	"strings"
)

const defaultTitle = "Pages in this section"

// Options controls one rendered subpages invocation.
type Options struct {
	// Title is the visible navigation heading when ShowTitle is true.
	Title string
	// ShowTitle controls whether the visible navigation heading is rendered.
	ShowTitle bool
}

// Parse recognizes one standalone {{subpages}} invocation.
func Parse(line string) (Options, bool) {
	value := strings.TrimSpace(line)
	if value == "{{subpages}}" {
		return Options{Title: defaultTitle, ShowTitle: true}, true
	}
	argument, ok := strings.CutPrefix(value, "{{subpages ")
	if !ok {
		return Options{}, false
	}
	argument, ok = strings.CutSuffix(argument, "}}")
	if !ok {
		return Options{}, false
	}
	argument = strings.TrimSpace(argument)
	name, encodedTitle, ok := strings.Cut(argument, "=")
	if !ok || strings.TrimSpace(name) != "title" {
		return Options{}, false
	}

	encodedTitle = strings.TrimSpace(encodedTitle)
	if len(encodedTitle) < 2 || encodedTitle[0] != '"' || encodedTitle[len(encodedTitle)-1] != '"' {
		return Options{}, false
	}

	title, err := strconv.Unquote(encodedTitle)
	if err != nil {
		return Options{}, false
	}

	return Options{Title: title, ShowTitle: title != ""}, true
}
