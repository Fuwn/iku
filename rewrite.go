package main

import "strings"

func (f *Formatter) rewrite(formattedSource []byte, lineInformationMap map[int]*lineInformation) []byte {
	sourceLines := strings.Split(string(formattedSource), "\n")
	resultLines := make([]string, 0, len(sourceLines))
	previousWasOpenBrace := false
	previousStatementType := ""
	previousWasComment := false
	previousWasTopLevel := false
	previousWasScoped := false
	insideRawString := false

	for lineIndex, currentLine := range sourceLines {
		backtickCount := countRawStringDelimiters(currentLine)
		wasInsideRawString := insideRawString

		if backtickCount%2 == 1 {
			insideRawString = !insideRawString
		}

		if wasInsideRawString {
			resultLines = append(resultLines, currentLine)

			continue
		}

		lineNumber := lineIndex + 1
		trimmedLine := strings.TrimSpace(currentLine)

		if trimmedLine == "" {
			continue
		}

		isClosingBrace := closingBracePattern.MatchString(currentLine)
		isOpeningBrace := openingBracePattern.MatchString(currentLine)
		isCaseLabel := caseLabelPattern.MatchString(currentLine)
		isCommentOnlyLine := isCommentOnly(currentLine)
		isPackageDeclaration := isPackageLine(trimmedLine)
		currentInformation := lineInformationMap[lineNumber]
		currentStatementType := ""

		if currentInformation != nil {
			currentStatementType = currentInformation.statementType
		}

		if isPackageDeclaration {
			currentStatementType = "package"
		}

		needsBlankLine := false
		currentIsTopLevel := currentInformation != nil && currentInformation.isTopLevel
		currentIsScoped := currentInformation != nil && currentInformation.isScoped

		if len(resultLines) > 0 && !previousWasOpenBrace && !isClosingBrace && !isCaseLabel {
			if currentIsTopLevel && previousWasTopLevel && currentStatementType != previousStatementType {
				if f.CommentMode == CommentsFollow && previousWasComment {
				} else {
					needsBlankLine = true
				}
			} else if currentInformation != nil && (currentIsScoped || previousWasScoped) {
				if f.CommentMode == CommentsFollow && previousWasComment {
				} else {
					needsBlankLine = true
				}
			} else if currentStatementType != "" && previousStatementType != "" && currentStatementType != previousStatementType {
				if f.CommentMode == CommentsFollow && previousWasComment {
				} else {
					needsBlankLine = true
				}
			}

			if f.CommentMode == CommentsFollow && isCommentOnlyLine && !previousWasComment {
				nextLineNumber := f.findNextNonCommentLine(sourceLines, lineIndex+1)

				if nextLineNumber > 0 {
					nextInformation := lineInformationMap[nextLineNumber]

					if nextInformation != nil {
						nextIsTopLevel := nextInformation.isTopLevel
						nextIsScoped := nextInformation.isScoped

						if nextIsTopLevel && previousWasTopLevel && nextInformation.statementType != previousStatementType {
							needsBlankLine = true
						} else if nextIsScoped || previousWasScoped {
							needsBlankLine = true
						} else if nextInformation.statementType != "" && previousStatementType != "" && nextInformation.statementType != previousStatementType {
							needsBlankLine = true
						}
					}
				}
			}
		}

		if needsBlankLine {
			resultLines = append(resultLines, "")
		}

		resultLines = append(resultLines, currentLine)
		previousWasOpenBrace = isOpeningBrace || isCaseLabel
		previousWasComment = isCommentOnlyLine

		if currentInformation != nil {
			previousStatementType = currentInformation.statementType
			previousWasTopLevel = currentInformation.isTopLevel
			previousWasScoped = currentInformation.isScoped
		} else if currentStatementType != "" {
			previousStatementType = currentStatementType
			previousWasTopLevel = false
			previousWasScoped = false
		}
	}

	outputString := strings.Join(resultLines, "\n")

	if !strings.HasSuffix(outputString, "\n") {
		outputString += "\n"
	}

	return []byte(outputString)
}

func (f *Formatter) findNextNonCommentLine(sourceLines []string, startLineIndex int) int {
	for lineIndex := startLineIndex; lineIndex < len(sourceLines); lineIndex++ {
		trimmedLine := strings.TrimSpace(sourceLines[lineIndex])

		if trimmedLine == "" {
			continue
		}

		if isCommentOnly(sourceLines[lineIndex]) {
			continue
		}

		return lineIndex + 1
	}

	return 0
}
