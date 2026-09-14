package main

import (
	chroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// curatedLexerNames keeps the bundled highlighter useful for common wiki,
// application, infrastructure, and configuration snippets without exposing
// every Chroma lexer to runtime sublexer lookup.
var curatedLexerNames = [...]string{
	"Bash",
	"C",
	"C++",
	"C#",
	"CSS",
	"Docker",
	"Go",
	"HCL",
	"HTML",
	"Java",
	"JavaScript",
	"JSON",
	"Kotlin",
	"Lua",
	"Makefile",
	"Markdown",
	"Nginx configuration file",
	"PHP",
	"PowerShell",
	"Python",
	"Ruby",
	"Rust",
	"SQL",
	"Terraform",
	"TOML",
	"TypeScript",
	"XML",
	"YAML",
	"Zig",
}

// newCuratedLexerRegistry reuses Chroma's maintained lexer definitions while
// giving selected lexers a small registry for aliases and nested lexer lookup.
// Chroma keeps compiled regex rules on these lexer instances after first use.
func newCuratedLexerRegistry() *chroma.LexerRegistry {
	registry := chroma.NewLexerRegistry()
	for _, name := range curatedLexerNames {
		lexer := lexers.Get(name)
		if lexer == nil {
			panic("syntax highlighting lexer is unavailable: " + name)
		}
		registry.Register(lexer)
	}

	return registry
}
