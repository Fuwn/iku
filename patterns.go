package main

import "regexp"

var (
	closingBracePattern = regexp.MustCompile(`^\s*[\}\)]`)
	openingBracePattern = regexp.MustCompile(`[\{\(]\s*$`)
	caseLabelPattern    = regexp.MustCompile(`^\s*(case\s|default\s*:)|(^\s+.*:\s*$)`)
)

func isCommentOnly(sourceLine string) bool {
	for characterIndex := range len(sourceLine) {
		character := sourceLine[characterIndex]

		if character == ' ' || character == '\t' {
			continue
		}

		return len(sourceLine) > characterIndex+1 && sourceLine[characterIndex] == '/' && sourceLine[characterIndex+1] == '/'
	}

	return false
}

func isPackageLine(trimmedLine string) bool {
	return len(trimmedLine) > 8 && trimmedLine[:8] == "package "
}

func countRawStringDelimiters(sourceLine string) int {
	delimiterCount := 0
	insideDoubleQuotedString := false
	insideCharacterLiteral := false

	for characterIndex := 0; characterIndex < len(sourceLine); characterIndex++ {
		character := sourceLine[characterIndex]

		if insideCharacterLiteral {
			if character == '\\' && characterIndex+1 < len(sourceLine) {
				characterIndex++

				continue
			}

			if character == '\'' {
				insideCharacterLiteral = false
			}

			continue
		}

		if insideDoubleQuotedString {
			if character == '\\' && characterIndex+1 < len(sourceLine) {
				characterIndex++

				continue
			}

			if character == '"' {
				insideDoubleQuotedString = false
			}

			continue
		}

		if character == '\'' {
			insideCharacterLiteral = true

			continue
		}

		if character == '"' {
			insideDoubleQuotedString = true

			continue
		}

		if character == '`' {
			delimiterCount++
		}
	}

	return delimiterCount
}
