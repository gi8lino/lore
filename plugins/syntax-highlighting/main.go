package main

import (
	"bytes"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/gi8lino/lore/pluginapi"
)

// main runs the package entry point.
func main() {}

// transform highlights one fenced code block when Chroma recognizes its language.
func transform(request pluginapi.RenderRequest) pluginapi.RenderResult {
	if !supportsRequest(request) {
		return pluginapi.RenderResult{Error: "unsupported syntax-highlighting request"}
	}

	language := strings.ToLower(strings.TrimSpace(request.Language))
	if language == "" {
		return pluginapi.RenderResult{}
	}

	lexer := lexers.Get(language)
	if lexer == nil {
		return pluginapi.RenderResult{}
	}

	iterator, err := lexer.Tokenise(nil, request.Source)
	if err != nil {
		return pluginapi.RenderResult{Error: "tokenize code: " + err.Error()}
	}

	style := styles.Get("github-dark")
	if style == nil {
		return pluginapi.RenderResult{Error: "syntax highlighting style is unavailable"}
	}

	var output bytes.Buffer
	formatter := chromahtml.New(chromahtml.WithClasses(true))
	if err := formatter.Format(&output, style, iterator); err != nil {
		return pluginapi.RenderResult{Error: "format highlighted code: " + err.Error()}
	}

	return pluginapi.RenderResult{
		Matched: true,
		Parts:   []pluginapi.RenderPart{{Text: output.String()}},
	}
}

// supportsRequest reports whether the request targets this highlighter module.
func supportsRequest(request pluginapi.RenderRequest) bool {
	return request.APIVersion == pluginapi.Version &&
		request.Module == "chroma" &&
		request.Stage == "highlight"
}
