package engine

import "strings"

type CommentMode int

const (
	CommentsFollow CommentMode = iota
	CommentsPrecede
	CommentsStandalone
)

type Engine struct {
	CommentMode           CommentMode
	GroupSingleLineScopes bool
}

func (e *Engine) format(events []LineEvent, resultBuilder *strings.Builder) {
	hasWrittenContent := false
	previousWasOpenBrace := false
	previousStatementType := ""
	previousWasComment := false
	previousWasTopLevel := false
	previousWasScoped := false
	previousWasSingleLineScope := false

	for eventIndex, event := range events {
		if event.InRawString {
			if hasWrittenContent {
				resultBuilder.WriteByte('\n')
			}

			resultBuilder.WriteString(event.Content)

			hasWrittenContent = true

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
		currentIsSingleLineScope := currentIsScoped && !event.IsOpeningBrace && !event.IsClosingBrace

		if hasWrittenContent && !previousWasOpenBrace && !event.IsClosingBrace && !event.IsCaseLabel && !event.IsContinuation {
			if currentIsTopLevel && previousWasTopLevel && currentStatementType != previousStatementType {
				if e.CommentMode != CommentsFollow || !previousWasComment {
					needsBlankLine = true
				}
			} else if event.HasASTInfo && (currentIsScoped || previousWasScoped) {
				if e.GroupSingleLineScopes && currentIsSingleLineScope && previousWasSingleLineScope && currentStatementType == previousStatementType {
					needsBlankLine = false
				} else if e.CommentMode != CommentsFollow || !previousWasComment {
					needsBlankLine = true
				}
			} else if currentStatementType != "" && previousStatementType != "" && currentStatementType != previousStatementType {
				if e.CommentMode != CommentsFollow || !previousWasComment {
					needsBlankLine = true
				}
			}

			if e.CommentMode == CommentsFollow && event.IsCommentOnly && !previousWasComment {
				nextIndex := e.findNextNonComment(events, eventIndex+1)

				if nextIndex >= 0 {
					nextNonCommentEvent := events[nextIndex]

					if nextNonCommentEvent.HasASTInfo {
						nextIsTopLevel := nextNonCommentEvent.IsTopLevel
						nextIsScoped := nextNonCommentEvent.IsScoped

						if nextIsTopLevel && previousWasTopLevel && nextNonCommentEvent.StatementType != previousStatementType {
							needsBlankLine = true
						} else if nextIsScoped || previousWasScoped {
							needsBlankLine = true
						} else if nextNonCommentEvent.StatementType != "" && previousStatementType != "" && nextNonCommentEvent.StatementType != previousStatementType {
							needsBlankLine = true
						}
					}
				}
			}
		}

		if needsBlankLine {
			resultBuilder.WriteByte('\n')
		}

		if hasWrittenContent {
			resultBuilder.WriteByte('\n')
		}

		resultBuilder.WriteString(event.Content)

		hasWrittenContent = true
		previousWasOpenBrace = event.IsOpeningBrace || event.IsCaseLabel
		previousWasComment = event.IsCommentOnly

		if event.HasASTInfo {
			previousStatementType = event.StatementType
			previousWasTopLevel = event.IsTopLevel
			previousWasScoped = event.IsScoped
			previousWasSingleLineScope = currentIsSingleLineScope
		} else if currentStatementType != "" {
			previousStatementType = currentStatementType
			previousWasTopLevel = false
			previousWasScoped = false
			previousWasSingleLineScope = false
		}
	}

	resultBuilder.WriteByte('\n')
}

func (e *Engine) FormatToString(events []LineEvent) string {
	var resultBuilder strings.Builder

	resultBuilder.Grow(len(events) * 40)
	e.format(events, &resultBuilder)

	return resultBuilder.String()
}

func (e *Engine) FormatToBytes(events []LineEvent) []byte {
	var resultBuilder strings.Builder

	resultBuilder.Grow(len(events) * 40)
	e.format(events, &resultBuilder)

	return []byte(resultBuilder.String())
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
