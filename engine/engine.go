package engine

import "strings"

type CommentMode int

const (
	CommentsFollow CommentMode = iota
	CommentsPrecede
	CommentsStandalone
)

type Engine struct {
	CommentMode CommentMode
}

func (e *Engine) Format(events []LineEvent) []string {
	resultLines := make([]string, 0, len(events))
	previousWasOpenBrace := false
	previousStatementType := ""
	previousWasComment := false
	previousWasTopLevel := false
	previousWasScoped := false

	for eventIndex, event := range events {
		if event.InRawString {
			resultLines = append(resultLines, event.Content)

			continue
		}

		if event.IsBlank {
			continue
		}

		currentStatementType := event.StatementType

		if event.IsPackageDecl {
			currentStatementType = "package"
		}

		needsBlankLine := false
		currentIsTopLevel := event.HasASTInfo && event.IsTopLevel
		currentIsScoped := event.HasASTInfo && event.IsScoped

		if len(resultLines) > 0 && !previousWasOpenBrace && !event.IsClosingBrace && !event.IsCaseLabel {
			if currentIsTopLevel && previousWasTopLevel && currentStatementType != previousStatementType {
				if !(e.CommentMode == CommentsFollow && previousWasComment) {
					needsBlankLine = true
				}
			} else if event.HasASTInfo && (currentIsScoped || previousWasScoped) {
				if !(e.CommentMode == CommentsFollow && previousWasComment) {
					needsBlankLine = true
				}
			} else if currentStatementType != "" && previousStatementType != "" && currentStatementType != previousStatementType {
				if !(e.CommentMode == CommentsFollow && previousWasComment) {
					needsBlankLine = true
				}
			}

			if e.CommentMode == CommentsFollow && event.IsCommentOnly && !previousWasComment {
				nextIndex := e.findNextNonComment(events, eventIndex+1)

				if nextIndex >= 0 {
					next := events[nextIndex]

					if next.HasASTInfo {
						nextIsTopLevel := next.IsTopLevel
						nextIsScoped := next.IsScoped

						if nextIsTopLevel && previousWasTopLevel && next.StatementType != previousStatementType {
							needsBlankLine = true
						} else if nextIsScoped || previousWasScoped {
							needsBlankLine = true
						} else if next.StatementType != "" && previousStatementType != "" && next.StatementType != previousStatementType {
							needsBlankLine = true
						}
					}
				}
			}
		}

		if needsBlankLine {
			resultLines = append(resultLines, "")
		}

		resultLines = append(resultLines, event.Content)
		previousWasOpenBrace = event.IsOpeningBrace || event.IsCaseLabel
		previousWasComment = event.IsCommentOnly

		if event.HasASTInfo {
			previousStatementType = event.StatementType
			previousWasTopLevel = event.IsTopLevel
			previousWasScoped = event.IsScoped
		} else if currentStatementType != "" {
			previousStatementType = currentStatementType
			previousWasTopLevel = false
			previousWasScoped = false
		}
	}

	return resultLines
}

func (e *Engine) FormatToString(events []LineEvent) string {
	lines := e.Format(events)
	output := strings.Join(lines, "\n")

	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return output
}

func (e *Engine) findNextNonComment(events []LineEvent, startIndex int) int {
	for eventIndex := startIndex; eventIndex < len(events); eventIndex++ {
		if events[eventIndex].IsBlank {
			continue
		}

		if events[eventIndex].IsCommentOnly {
			continue
		}

		return eventIndex
	}

	return -1
}
