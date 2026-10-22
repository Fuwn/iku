package main

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ecmaScriptLexicalState struct {
	quote              byte
	insideBlockComment bool
	insideRegexClass   bool
}

func (state *ecmaScriptLexicalState) maskLiteralsAndComments(sourceLine string) string {
	if state.insideBlockComment {
		commentEnd := strings.Index(sourceLine, "*/")

		if commentEnd < 0 {
			return ""
		}

		sourceLine = sourceLine[commentEnd+2:]
		state.insideBlockComment = false
	}

	if state.quote == 0 && !strings.ContainsAny(sourceLine, "\"'`/") {
		return sourceLine
	}

	code := []byte(sourceLine)

	for characterIndex := 0; characterIndex < len(code); characterIndex++ {
		character := code[characterIndex]

		if state.quote != 0 {
			code[characterIndex] = ' '

			if character == '\\' && characterIndex+1 < len(code) {
				code[characterIndex+1] = ' '
				characterIndex++
			} else if state.quote == '/' && character == '[' {
				state.insideRegexClass = true
			} else if state.quote == '/' && character == ']' {
				state.insideRegexClass = false
			} else if character == state.quote && !state.insideRegexClass {
				state.quote = 0
			}

			continue
		}

		if character == '/' && characterIndex+1 < len(code) {
			nextCharacter := code[characterIndex+1]

			if nextCharacter == '/' {
				return string(code[:characterIndex])
			}

			if nextCharacter == '*' {
				code[characterIndex] = ' '
				code[characterIndex+1] = ' '

				for characterIndex += len("/*"); characterIndex < len(code); characterIndex++ {
					character := code[characterIndex]

					code[characterIndex] = ' '

					if character == '*' && characterIndex+1 < len(code) && code[characterIndex+1] == '/' {
						code[characterIndex+1] = ' '
						characterIndex++

						break
					}
				}

				state.insideBlockComment = characterIndex >= len(code)

				continue
			}
		}

		if character == '\'' || character == '"' || character == '`' {
			state.quote = character
			code[characterIndex] = '0'
		} else if character == '/' && ecmaScriptStartsRegex(code[:characterIndex]) {
			state.quote = '/'
			code[characterIndex] = '0'
		}
	}

	return string(code)
}

func ecmaScriptStartsRegex(prefix []byte) bool {
	prefix = bytes.TrimSpace(prefix)

	if len(prefix) == 0 || strings.ContainsRune("=([{,:;!?&|+-*%^~<>", rune(prefix[len(prefix)-1])) {
		return true
	}

	// Check suffixes to avoid rescanning the expression for every slash.
	for _, keyword := range []string{"return", "throw", "yield", "case", "typeof", "void", "delete"} {
		if bytes.HasSuffix(prefix, []byte(keyword)) {
			precedingCharacter, _ := utf8.DecodeLastRune(prefix[:len(prefix)-len(keyword)])

			return len(prefix) == len(keyword) || unicode.IsSpace(precedingCharacter)
		}
	}

	return false
}
