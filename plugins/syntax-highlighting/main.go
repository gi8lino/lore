package main

import (
	"bytes"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/gi8lino/lore/pluginapi"
	"github.com/gi8lino/lore/pluginsdk"
)

var (
	highlightFormatter = chromahtml.New(chromahtml.WithClasses(true))
	highlightStyle     = styles.Get("github-dark")
	highlightLexers    = newCuratedLexerRegistry()
)

func main() {}

func init() {
	if highlightStyle == nil {
		panic("syntax highlighting style is unavailable")
	}
	pluginsdk.RegisterModule("chroma", transform)
}

// transform highlights one fenced code block using its explicit Markdown
// fence language. Chroma compiles each selected lexer lazily and retains the
// compiled rules on the curated registry instance for subsequent calls.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if !supportsRequest(request) {
		return pluginapi.RenderResult{Error: "unsupported syntax-highlighting request"}
	}

	lexer := highlightLexers.Get(strings.TrimSpace(request.Language))
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
