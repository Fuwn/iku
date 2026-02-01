package main

func isWhitespace(character byte) bool {
	return character == ' ' || character == '\t' || character == '\n' || character == '\r' || character == '\f'
}

func isClosingBrace(sourceLine string) bool {
	for characterIndex := 0; characterIndex < len(sourceLine); characterIndex++ {
		character := sourceLine[characterIndex]

		if isWhitespace(character) {
			continue
		}

		return character == '}' || character == ')'
	}

	return false
}

func isOpeningBrace(sourceLine string) bool {
	for characterIndex := len(sourceLine) - 1; characterIndex >= 0; characterIndex-- {
		character := sourceLine[characterIndex]

		if isWhitespace(character) {
			continue
		}

		return character == '{' || character == '('
	}

	return false
}

func isCaseLabel(sourceLine string) bool {
	firstNonWhitespaceIndex := 0

	for firstNonWhitespaceIndex < len(sourceLine) && isWhitespace(sourceLine[firstNonWhitespaceIndex]) {
		firstNonWhitespaceIndex++
	}

	if firstNonWhitespaceIndex >= len(sourceLine) {
		return false
	}

	contentAfterWhitespace := sourceLine[firstNonWhitespaceIndex:]

	if len(contentAfterWhitespace) >= 5 && contentAfterWhitespace[:4] == "case" && isWhitespace(contentAfterWhitespace[4]) {
		return true
	}

	if len(contentAfterWhitespace) >= 7 && contentAfterWhitespace[:7] == "default" {
		for characterIndex := 7; characterIndex < len(contentAfterWhitespace); characterIndex++ {
			character := contentAfterWhitespace[characterIndex]

			if isWhitespace(character) {
				continue
			}

			if character == ':' {
				return true
			}

			break
		}
	}

	if firstNonWhitespaceIndex > 0 {
		lastNonWhitespaceIndex := len(sourceLine) - 1

		for lastNonWhitespaceIndex >= 0 && isWhitespace(sourceLine[lastNonWhitespaceIndex]) {
			lastNonWhitespaceIndex--
		}

		if lastNonWhitespaceIndex >= 0 && sourceLine[lastNonWhitespaceIndex] == ':' {
			return true
		}
	}

	return false
}

func isCommentOnly(sourceLine string) bool {
	for characterIndex := range len(sourceLine) {
		character := sourceLine[characterIndex]

		if isWhitespace(character) {
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
