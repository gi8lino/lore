package main

import (
	"html"
	"strings"
)

type token struct {
	class string
	text  string
}

type language struct {
	name    string
	aliases []string
	scan    func(string) []token
}

var languages = buildLanguages()

func lookupLanguage(name string) (language, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	language, ok := languages[name]
	return language, ok
}

func buildLanguages() map[string]language {
	registry := make(map[string]language)
	register := func(item language) {
		for _, alias := range item.aliases {
			key := strings.ToLower(alias)
			if _, exists := registry[key]; exists {
				panic("duplicate syntax-highlighting alias: " + key)
			}
			registry[key] = item
		}
	}

	for _, item := range languageDefinitions() {
		register(item)
	}
	return registry
}

func renderHighlighted(requested, source string, language language) string {
	var output strings.Builder
	output.Grow(len(source) + len(source)/4 + 64)
	output.WriteString(`<pre class="chroma"><code class="language-`)
	output.WriteString(html.EscapeString(strings.ToLower(strings.TrimSpace(requested))))
	output.WriteString(`">`)
	for _, item := range language.scan(source) {
		if item.class == "" {
			output.WriteString(html.EscapeString(item.text))
			continue
		}
		output.WriteString(`<span class="`)
		output.WriteString(item.class)
		output.WriteString(`">`)
		output.WriteString(html.EscapeString(item.text))
		output.WriteString(`</span>`)
	}
	output.WriteString("</code></pre>\n")
	return output.String()
}

func appendToken(tokens []token, class, text string) []token {
	if text == "" {
		return tokens
	}
	if len(tokens) != 0 && tokens[len(tokens)-1].class == class {
		tokens[len(tokens)-1].text += text
		return tokens
	}
	return append(tokens, token{class: class, text: text})
}
