package main

import "strings"

func isWhitespace(character byte) bool {
	return character == ' ' || character == '\t' || character == '\n' || character == '\r' || character == '\f'
}

func trimLeadingWhitespace(source string) string {
	start := 0

	for start < len(source) && isWhitespace(source[start]) {
		start++
	}

	return source[start:]
}

func trimTrailingWhitespace(source string) string {
	end := len(source)

	for end > 0 && isWhitespace(source[end-1]) {
		end--
	}

	return source[:end]
}

func isClosingBrace(sourceLine string) bool {
	content := trimLeadingWhitespace(sourceLine)

	return len(content) > 0 && (content[0] == '}' || content[0] == ')')
}

func isOpeningBrace(sourceLine string) bool {
	content := trimTrailingWhitespace(sourceLine)

	return strings.HasSuffix(content, "{") || strings.HasSuffix(content, "(")
}
