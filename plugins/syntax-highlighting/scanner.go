package main

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type codeProfile struct {
	caseInsensitive  bool
	lineComments     []string
	blockComments    []commentPair
	quotes           []quoteRule
	keywords         map[string]string
	builtins         map[string]struct{}
	variablePrefixes string
	identifierExtra  string
	keySeparator     byte
	atKeyword        bool
}

type commentPair struct {
	start string
	end   string
}

type quoteRule struct {
	start  string
	end    string
	escape bool
	class  string
}

func scanCode(source string, profile codeProfile) []token {
	tokens := make([]token, 0, 32)
	for position := 0; position < len(source); {
		if end, ok := scanComment(source, position, profile); ok {
			tokens = appendToken(tokens, "c1", source[position:end])
			position = end
			continue
		}
		if end, class, ok := scanQuoted(source, position, profile.quotes); ok {
			tokens = appendToken(tokens, class, source[position:end])
			position = end
			continue
		}

		current := source[position]
		if strings.ContainsRune(profile.variablePrefixes, rune(current)) && position+1 < len(source) && isIdentifierStart(source[position+1], profile.identifierExtra) {
			end := scanIdentifier(source, position+1, profile.identifierExtra)
			tokens = appendToken(tokens, "nv", source[position:end])
			position = end
			continue
		}
		if current == '@' && profile.atKeyword && position+1 < len(source) && isIdentifierStart(source[position+1], profile.identifierExtra) {
			end := scanIdentifier(source, position+1, profile.identifierExtra)
			tokens = appendToken(tokens, "k", source[position:end])
			position = end
			continue
		}
		if isNumberStart(source, position) {
			end, class := scanNumber(source, position)
			tokens = appendToken(tokens, class, source[position:end])
			position = end
			continue
		}
		if isIdentifierStart(current, profile.identifierExtra) {
			end := scanIdentifier(source, position, profile.identifierExtra)
			word := source[position:end]
			key := word
			if profile.caseInsensitive {
				key = strings.ToLower(word)
			}
			class := profile.keywords[key]
			if class == "" {
				if _, ok := profile.builtins[key]; ok {
					class = "nb"
				} else if profile.keySeparator != 0 && nextNonSpace(source, end) == profile.keySeparator {
					class = "na"
				} else if nextNonSpace(source, end) == '(' {
					class = "nf"
				}
			}
			tokens = appendToken(tokens, class, word)
			position = end
			continue
		}
		if isOperator(current) {
			end := scanOperator(source, position)
			tokens = appendToken(tokens, "o", source[position:end])
			position = end
			continue
		}
		if isPunctuation(current) {
			tokens = appendToken(tokens, "p", source[position:position+1])
			position++
			continue
		}

		_, size := utf8.DecodeRuneInString(source[position:])
		if size <= 0 {
			size = 1
		}
		tokens = appendToken(tokens, "", source[position:position+size])
		position += size
	}
	return tokens
}

func scanComment(source string, position int, profile codeProfile) (int, bool) {
	for _, pair := range profile.blockComments {
		if strings.HasPrefix(source[position:], pair.start) {
			if end := strings.Index(source[position+len(pair.start):], pair.end); end >= 0 {
				return position + len(pair.start) + end + len(pair.end), true
			}
			return len(source), true
		}
	}
	for _, marker := range profile.lineComments {
		if strings.HasPrefix(source[position:], marker) {
			if end := strings.IndexByte(source[position:], '\n'); end >= 0 {
				return position + end, true
			}
			return len(source), true
		}
	}
	return position, false
}

func scanQuoted(source string, position int, rules []quoteRule) (int, string, bool) {
	for _, rule := range rules {
		if !strings.HasPrefix(source[position:], rule.start) {
			continue
		}
		for current := position + len(rule.start); current < len(source); {
			if rule.escape && source[current] == '\\' {
				if current+1 < len(source) {
					current += 2
					continue
				}
				return len(source), rule.class, true
			}
			if strings.HasPrefix(source[current:], rule.end) {
				return current + len(rule.end), rule.class, true
			}
			_, size := utf8.DecodeRuneInString(source[current:])
			if size <= 0 {
				size = 1
			}
			current += size
		}
		return len(source), rule.class, true
	}
	return position, "", false
}

func scanIdentifier(source string, position int, extra string) int {
	for position < len(source) {
		current := source[position]
		if current < utf8.RuneSelf {
			if !isIdentifierPart(current, extra) {
				break
			}
			position++
			continue
		}
		r, size := utf8.DecodeRuneInString(source[position:])
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			break
		}
		position += size
	}
	return position
}

func isIdentifierStart(value byte, extra string) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= utf8.RuneSelf || strings.IndexByte(extra, value) >= 0
}

func isIdentifierPart(value byte, extra string) bool {
	return isIdentifierStart(value, extra) || value >= '0' && value <= '9'
}

func isNumberStart(source string, position int) bool {
	current := source[position]
	if current >= '0' && current <= '9' {
		return true
	}
	return current == '.' && position+1 < len(source) && source[position+1] >= '0' && source[position+1] <= '9'
}

func scanNumber(source string, position int) (int, string) {
	start := position
	class := "mi"
	if source[position] == '.' {
		class = "mf"
		position++
	}
	if position+1 < len(source) && source[position] == '0' {
		switch source[position+1] {
		case 'x', 'X':
			position += 2
			for position < len(source) && (isHex(source[position]) || source[position] == '_') {
				position++
			}
			return position, "mh"
		case 'b', 'B', 'o', 'O':
			position += 2
			for position < len(source) && (source[position] >= '0' && source[position] <= '9' || source[position] == '_') {
				position++
			}
			return position, "mi"
		}
	}
	for position < len(source) {
		current := source[position]
		switch {
		case current >= '0' && current <= '9', current == '_':
			position++
		case current == '.':
			class = "mf"
			position++
		case current == 'e' || current == 'E':
			class = "mf"
			position++
			if position < len(source) && (source[position] == '+' || source[position] == '-') {
				position++
			}
		default:
			if position == start {
				position++
			}
			return position, class
		}
	}
	return position, class
}

func nextNonSpace(source string, position int) byte {
	for position < len(source) {
		switch source[position] {
		case ' ', '\t', '\r', '\n':
			position++
		default:
			return source[position]
		}
	}
	return 0
}

func isHex(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}

func isOperator(value byte) bool    { return strings.IndexByte("+-*/%=&|!<>^~?:", value) >= 0 }
func isPunctuation(value byte) bool { return strings.IndexByte("()[]{}.,;", value) >= 0 }

func scanOperator(source string, position int) int {
	end := position + 1
	for end < len(source) && isOperator(source[end]) && end-position < 3 {
		end++
	}
	return end
}
