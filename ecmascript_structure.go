package main

import (
	"bytes"
	"github.com/Fuwn/iku/engine"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ecmaScriptDelimiter struct {
	closing          byte
	isBody           bool
	openingLine      int
	statementStart   int
	currentStatement int
}

type ecmaScriptLexicalState struct {
	quote              byte
	insideBlockComment bool
	insideRegexClass   bool
}

func ecmaScriptHasFunctionKeyword(code string) bool {
	for searchStart := 0; searchStart < len(code); {
		relativeIndex := strings.Index(code[searchStart:], "function")

		if relativeIndex < 0 {
			return false
		}

		keywordStart := searchStart + relativeIndex
		keywordEnd := keywordStart + len("function")
		validStart := keywordStart == 0 || strings.ContainsRune(" \t=(:,;[{!", rune(code[keywordStart-1]))
		validEnd := keywordEnd == len(code) || strings.ContainsRune(" \t(*", rune(code[keywordEnd]))

		if validStart && validEnd {
			return true
		}

		searchStart = keywordEnd
	}

	return false
}

func ecmaScriptBraceIsBody(prefix string, statementStart int, events []engine.LineEvent, delimiters []ecmaScriptDelimiter) bool {
	if strings.HasSuffix(prefix, ")") || strings.HasSuffix(prefix, "=>") {
		return true
	}

	for _, keyword := range []string{"else", "try", "finally", "do"} {
		if prefix == keyword || strings.HasSuffix(prefix, " "+keyword) {
			return true
		}
	}

	if statementStart < 0 || (len(delimiters) > 0 && !delimiters[len(delimiters)-1].isBody) {
		return false
	}

	switch events[statementStart].StatementType {
	case "function", "type", "namespace":
		return true
	default:
		return prefix == ""
	}
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

func ecmaScriptHasAssignment(code string) bool {
	parenthesisDepth := 0
	bracketDepth := 0

	for characterIndex := 0; characterIndex < len(code); characterIndex++ {
		character := code[characterIndex]

		switch character {
		case '(':
			parenthesisDepth++
		case ')':
			parenthesisDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case '=':
			if parenthesisDepth != 0 || bracketDepth != 0 {
				continue
			}

			previousCharacter := byte(0)
			nextCharacter := byte(0)

			if characterIndex > 0 {
				previousCharacter = code[characterIndex-1]
			}

			if characterIndex+1 < len(code) {
				nextCharacter = code[characterIndex+1]
			}

			isComparison := previousCharacter == '=' || previousCharacter == '!' ||
				((previousCharacter == '<' || previousCharacter == '>') && (characterIndex < 2 || code[characterIndex-2] != previousCharacter))

			if !isComparison && nextCharacter != '=' && nextCharacter != '>' {
				return true
			}
		case '+', '-':
			if parenthesisDepth == 0 && bracketDepth == 0 && characterIndex+1 < len(code) && code[characterIndex+1] == character {
				return true
			}
		}
	}

	return false
}
