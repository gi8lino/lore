package main

import "strings"

func scanMarkdown(source string) []token {
	tokens := make([]token, 0, 24)
	for start := 0; start < len(source); {
		end := strings.IndexByte(source[start:], '\n')
		if end < 0 {
			end = len(source)
		} else {
			end += start + 1
		}
		line := source[start:end]
		tokens = append(tokens, scanMarkdownLine(line)...)
		start = end
	}
	return tokens
}

func scanMarkdownLine(line string) []token {
	tokens := make([]token, 0, 8)
	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	if indent != 0 {
		tokens = appendToken(tokens, "", line[:indent])
	}
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
		return appendToken(tokens, "cp", trimmed)
	}
	if strings.HasPrefix(trimmed, "#") {
		end := 0
		for end < len(trimmed) && trimmed[end] == '#' {
			end++
		}
		if end < len(trimmed) && trimmed[end] == ' ' {
			tokens = appendToken(tokens, "k", trimmed[:end])
			tokens = appendToken(tokens, "", trimmed[end:])
			return tokens
		}
	}
	if strings.HasPrefix(trimmed, "> ") {
		tokens = appendToken(tokens, "p", ">")
		trimmed = trimmed[1:]
	}
	if len(trimmed) >= 2 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') && trimmed[1] == ' ' {
		tokens = appendToken(tokens, "p", trimmed[:1])
		trimmed = trimmed[1:]
	}
	for len(trimmed) != 0 {
		open := strings.IndexByte(trimmed, '`')
		if open < 0 {
			tokens = appendToken(tokens, "", trimmed)
			break
		}
		tokens = appendToken(tokens, "", trimmed[:open])
		close := strings.IndexByte(trimmed[open+1:], '`')
		if close < 0 {
			tokens = appendToken(tokens, "s2", trimmed[open:])
			break
		}
		close += open + 2
		tokens = appendToken(tokens, "s2", trimmed[open:close])
		trimmed = trimmed[close:]
	}
	return tokens
}
