package main

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

type rustLexicalState struct {
	insideString      bool
	insideRawString   bool
	rawStringHashes   int
	blockCommentDepth int
}

func (state *rustLexicalState) isInsideLiteral() bool {
	return state.insideString || state.insideRawString
}

// Strings, raw strings, and character literals become a placeholder operand followed by spaces, and
// comments become spaces, so structural scans see only code. Lifetimes and labels are kept.
func (state *rustLexicalState) maskLiteralsAndComments(sourceLine string) string {
	if !state.isInsideLiteral() && state.blockCommentDepth == 0 && !strings.ContainsAny(sourceLine, "\"'/") {
		return sourceLine
	}

	code := []byte(sourceLine)

	for characterIndex := 0; characterIndex < len(code); characterIndex++ {
		switch {
		case state.blockCommentDepth > 0:
			characterIndex = state.maskBlockComment(code, characterIndex)
		case state.insideString:
			characterIndex = state.maskString(code, characterIndex)
		case state.insideRawString:
			characterIndex = state.maskRawString(code, characterIndex)
		default:
			character := code[characterIndex]

			switch {
			case character == '/' && characterIndex+1 < len(code) && code[characterIndex+1] == '/':
				return string(code[:characterIndex])
			case character == '/' && characterIndex+1 < len(code) && code[characterIndex+1] == '*':
				state.blockCommentDepth = 1
				code[characterIndex] = ' '
				code[characterIndex+1] = ' '
				characterIndex = state.maskBlockComment(code, characterIndex+2)
			case character == '"':
				state.insideString = true
				code[characterIndex] = '0'
				characterIndex = state.maskString(code, characterIndex+1)
			case character == '\'':
				characterIndex = maskRustCharacterLiteral(code, characterIndex)
			case character == 'r' && rustStartsRawString(code, characterIndex):
				hashCount := 0

				for code[characterIndex+1+hashCount] == '#' {
					hashCount++
				}

				state.insideRawString = true
				state.rawStringHashes = hashCount
				code[characterIndex] = '0'

				for maskedIndex := characterIndex + 1; maskedIndex <= characterIndex+1+hashCount; maskedIndex++ {
					code[maskedIndex] = ' '
				}

				characterIndex = state.maskRawString(code, characterIndex+2+hashCount)
			}
		}
	}

	return string(code)
}

// Each mask method blanks code from start through the literal's or comment's end and returns the
// index of its final character, leaving the state open when the line ends first.
func (state *rustLexicalState) maskBlockComment(code []byte, start int) int {
	characterIndex := start

	for ; characterIndex < len(code); characterIndex++ {
		pairStartsHere := characterIndex+1 < len(code)

		switch {
		case pairStartsHere && code[characterIndex] == '/' && code[characterIndex+1] == '*':
			state.blockCommentDepth++
		case pairStartsHere && code[characterIndex] == '*' && code[characterIndex+1] == '/':
			state.blockCommentDepth--
		default:
			code[characterIndex] = ' '

			continue
		}

		code[characterIndex] = ' '
		code[characterIndex+1] = ' '
		characterIndex++

		if state.blockCommentDepth == 0 {
			return characterIndex
		}
	}

	return characterIndex
}

func (state *rustLexicalState) maskString(code []byte, start int) int {
	characterIndex := start

	for ; characterIndex < len(code); characterIndex++ {
		character := code[characterIndex]

		code[characterIndex] = ' '

		if character == '\\' && characterIndex+1 < len(code) {
			code[characterIndex+1] = ' '
			characterIndex++
		} else if character == '"' {
			state.insideString = false

			return characterIndex
		}
	}

	return characterIndex
}

func (state *rustLexicalState) maskRawString(code []byte, start int) int {
	characterIndex := start

	for ; characterIndex < len(code); characterIndex++ {
		closesString := code[characterIndex] == '"' && rustHashesFollow(code, characterIndex+1, state.rawStringHashes)

		code[characterIndex] = ' '

		if closesString {
			for maskedIndex := characterIndex + 1; maskedIndex <= characterIndex+state.rawStringHashes; maskedIndex++ {
				code[maskedIndex] = ' '
			}

			state.insideRawString = false

			return characterIndex + state.rawStringHashes
		}
	}

	return characterIndex
}

func rustHashesFollow(code []byte, start int, hashCount int) bool {
	if start+hashCount > len(code) {
		return false
	}

	for _, character := range code[start : start+hashCount] {
		if character != '#' {
			return false
		}
	}

	return true
}

// A quote begins a character literal when an escape or a single character and a closing quote
// follow it; otherwise it begins a lifetime or label, which stays visible.
func maskRustCharacterLiteral(code []byte, start int) int {
	literalEnd := -1

	switch {
	case start+3 < len(code) && code[start+1] == '\\':
		escapeEnd := min(len(code), start+3+len(`u{10FFFF}`))

		if closingOffset := bytes.IndexByte(code[start+3:escapeEnd], '\''); closingOffset >= 0 {
			literalEnd = start + 3 + closingOffset
		}
	case start+1 < len(code):
		_, characterWidth := utf8.DecodeRune(code[start+1:])

		if closingIndex := start + 1 + characterWidth; closingIndex < len(code) && code[closingIndex] == '\'' {
			literalEnd = closingIndex
		}
	}

	if literalEnd < 0 {
		return start
	}

	code[start] = '0'

	for maskedIndex := start + 1; maskedIndex <= literalEnd; maskedIndex++ {
		code[maskedIndex] = ' '
	}

	return literalEnd
}

// Raw strings begin with r, br, or cr outside an identifier, then any number of hashes and a quote.
// The same prefix without a quote is a raw identifier.
func rustStartsRawString(code []byte, rawPrefixIndex int) bool {
	tokenStart := rawPrefixIndex

	if tokenStart > 0 && (code[tokenStart-1] == 'b' || code[tokenStart-1] == 'c') {
		tokenStart--
	}

	if tokenStart > 0 && isRustIdentifierCharacter(code[tokenStart-1]) {
		return false
	}

	hashEnd := rawPrefixIndex + 1

	for hashEnd < len(code) && code[hashEnd] == '#' {
		hashEnd++
	}

	return hashEnd < len(code) && code[hashEnd] == '"'
}

func isRustIdentifierCharacter(character byte) bool {
	return character == '_' || character >= 0x80 || ('0' <= character && character <= '9') ||
		('a' <= character && character <= 'z') || ('A' <= character && character <= 'Z')
}
