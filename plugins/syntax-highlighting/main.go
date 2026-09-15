package main

import (
	"bytes"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/gi8lino/lore/pluginapi"
	"github.com/gi8lino/lore/pluginsdk"
)

var (
	highlightFormatter = chromahtml.New(chromahtml.WithClasses(true))
	highlightStyle     = styles.Get("github-dark")
	highlightLanguages = func() map[string]struct{} {
		names := lexers.Names(true)
		languages := make(map[string]struct{}, len(names))
		for _, name := range names {
			languages[strings.ToLower(name)] = struct{}{}
		}
		return languages
	}()
)

func main() {}

func init() {
	if highlightStyle == nil {
		panic("syntax highlighting style is unavailable")
	}
	pluginsdk.RegisterModule("chroma", transform)
}

// transform highlights one fenced code block using its explicit Markdown
// fence language. Chroma's global registry provides the complete maintained
// lexer set; Lore does not auto-detect a language from the source text.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if !supportsRequest(request) {
		return pluginapi.RenderResult{Error: "unsupported syntax-highlighting request"}
	}

	language := strings.ToLower(strings.TrimSpace(request.Language))
	if language == "" {
		return pluginapi.RenderResult{}
	}
	if _, ok := highlightLanguages[language]; !ok {
		return pluginapi.RenderResult{}
	}

	// Get selects only the explicitly supplied Chroma name or alias. Do not use
	// Match or Analyse here: fenced code is never language-detected from content.
	lexer := lexers.Get(language)
	if lexer == nil {
		return pluginapi.RenderResult{}
	}

	iterator, err := lexer.Tokenise(nil, request.Source)
	if err != nil {
		return pluginapi.RenderResult{Error: "tokenize code: " + err.Error()}
	}

	var output bytes.Buffer
	if err := highlightFormatter.Format(&output, highlightStyle, iterator); err != nil {
		return pluginapi.RenderResult{Error: "format highlighted code: " + err.Error()}
	}

	return pluginapi.RenderResult{Matched: true, Parts: []pluginapi.RenderPart{{Text: output.String()}}}
}

func supportsRequest(request pluginapi.RenderRequest) bool {
	return request.Module == "chroma" && request.Stage == "highlight"
}
