package main

import (
	"strings"
	"unicode/utf8"
)

func scanMarkup(source string) []token {
	tokens := make([]token, 0, 32)
	for position := 0; position < len(source); {
		if strings.HasPrefix(source[position:], "<!--") {
			end := strings.Index(source[position+4:], "-->")
			if end < 0 {
				end = len(source)
			} else {
				end = position + 4 + end + 3
			}
			tokens = appendToken(tokens, "c1", source[position:end])
			position = end
			continue
		}
		if strings.HasPrefix(source[position:], "<![CDATA[") {
			end := strings.Index(source[position+9:], "]]>")
			if end < 0 {
				end = len(source)
			} else {
				end = position + 9 + end + 3
			}
			tokens = appendToken(tokens, "s2", source[position:end])
			position = end
			continue
		}
		if source[position] != '<' {
			end := strings.IndexByte(source[position:], '<')
			if end < 0 {
				end = len(source)
			} else {
				end += position
			}
			tokens = appendToken(tokens, "", source[position:end])
			position = end
			continue
		}

		tokens = appendToken(tokens, "p", "<")
		position++
		if position < len(source) && (source[position] == '/' || source[position] == '!' || source[position] == '?') {
			tokens = appendToken(tokens, "p", source[position:position+1])
			position++
		}
		for position < len(source) && isMarkupSpace(source[position]) {
			tokens = appendToken(tokens, "", source[position:position+1])
			position++
		}
		if position < len(source) && isMarkupNameStart(source[position]) {
			end := scanMarkupName(source, position)
			tokens = appendToken(tokens, "nt", source[position:end])
			position = end
		}

		for position < len(source) && source[position] != '>' {
			if strings.HasPrefix(source[position:], "/>") {
				tokens = appendToken(tokens, "p", "/>")
				position += 2
				break
			}
			if isMarkupSpace(source[position]) {
				start := position
				for position < len(source) && isMarkupSpace(source[position]) {
					position++
				}
				tokens = appendToken(tokens, "", source[start:position])
				continue
			}
			if isMarkupNameStart(source[position]) {
				end := scanMarkupName(source, position)
				tokens = appendToken(tokens, "na", source[position:end])
				position = end
				continue
			}
			if source[position] == '=' {
				tokens = appendToken(tokens, "o", "=")
				position++
				continue
			}
			if source[position] == '"' || source[position] == '\'' {
				quote := source[position]
				end := position + 1
				for end < len(source) && source[end] != quote {
					if source[end] == '\\' && end+1 < len(source) {
						end += 2
					} else {
						_, size := utf8.DecodeRuneInString(source[end:])
						if size <= 0 {
							size = 1
						}
						end += size
					}
				}
				if end < len(source) {
					end++
				}
				tokens = appendToken(tokens, "s2", source[position:end])
				position = end
				continue
			}
			tokens = appendToken(tokens, "p", source[position:position+1])
			position++
		}
		if position < len(source) && source[position] == '>' {
			tokens = appendToken(tokens, "p", ">")
			position++
		}
	}
	return tokens
}

func isMarkupSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}
func isMarkupNameStart(value byte) bool {
	return value == '_' || value == ':' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}
func scanMarkupName(source string, position int) int {
	for position < len(source) {
		value := source[position]
		if !(isMarkupNameStart(value) || value == '-' || value == '.' || value >= '0' && value <= '9') {
			break
		}
		position++
	}
	return position
}
