// Package main implements the Subpages plugin.
package main

import (
	"strconv"
	"strings"
)

const defaultTitle = "Pages in this section"

// macroOptions controls one rendered subpages invocation.
type macroOptions struct {
	// Title is the visible navigation heading when ShowTitle is true.
	Title string
	// ShowTitle controls whether the visible navigation heading is rendered.
	ShowTitle bool
}

// parse recognizes one standalone {{subpages}} invocation.
func parse(line string) (macroOptions, bool) {
	value := strings.TrimSpace(line)
	if value == "{{subpages}}" {
		return macroOptions{Title: defaultTitle, ShowTitle: true}, true
	}
	argument, ok := strings.CutPrefix(value, "{{subpages ")
	if !ok {
		return macroOptions{}, false
	}
	argument, ok = strings.CutSuffix(argument, "}}")
	if !ok {
		return macroOptions{}, false
	}
	argument = strings.TrimSpace(argument)
	name, encodedTitle, ok := strings.Cut(argument, "=")
	if !ok || strings.TrimSpace(name) != "title" {
		return macroOptions{}, false
	}

	encodedTitle = strings.TrimSpace(encodedTitle)
	if !isQuoted(encodedTitle) {
		return macroOptions{}, false
	}

	title, err := strconv.Unquote(encodedTitle)
	if err != nil {
		return macroOptions{}, false
	}

	return macroOptions{Title: title, ShowTitle: title != ""}, true
}

// isQuoted reports whether value is enclosed in double quotes.
func isQuoted(value string) bool {
	return len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"'
}
