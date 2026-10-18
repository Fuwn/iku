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

	for _, event := range events {
		if event.InRawString {
			if hasWrittenContent {
				resultBuilder.WriteByte('\n')
			}

			resultBuilder.WriteString(event.Content)

			hasWrittenContent = true
			previousWasOpenBrace = event.IsOpeningBrace || event.IsCaseLabel

			if event.HasASTInfo {
				previousStatementType = event.StatementType
				previousWasTopLevel = event.IsTopLevel
				previousWasScoped = event.IsScoped
				previousWasSingleLineScope = event.IsSingleLineScope
				previousWasComment = event.IsCommentOnly
			}

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
		currentIsSingleLineScope := currentIsScoped && event.IsSingleLineScope
		sameStatementType := currentStatementType == previousStatementType
		groupedBindings := sameStatementType && currentStatementType == "binding" && !currentIsTopLevel && !previousWasTopLevel
		groupedCalls := sameStatementType && currentStatementType == "call"
		isDefinition := currentStatementType == "function" || currentStatementType == "func" || currentStatementType == "type"

		if hasWrittenContent && !previousWasOpenBrace && !event.IsClosingBrace && !event.IsCaseLabel && !event.IsContinuation {
			if currentIsTopLevel && previousWasTopLevel && currentStatementType != previousStatementType {
				if e.CommentMode != CommentsFollow || !previousWasComment {
					needsBlankLine = true
				}
			} else if groupedBindings || groupedCalls {
				needsBlankLine = false
			} else if event.HasASTInfo && (currentIsScoped || previousWasScoped) {
				if e.GroupSingleLineScopes && isDefinition && currentIsSingleLineScope && previousWasSingleLineScope && sameStatementType {
					needsBlankLine = false
				} else if e.CommentMode != CommentsFollow || !previousWasComment {
					needsBlankLine = true
				}
			} else if currentStatementType != "" && previousStatementType != "" && currentStatementType != previousStatementType {
				if e.CommentMode != CommentsFollow || !previousWasComment {
					needsBlankLine = true
				}
			}

			if event.IsCommentOnly {
				needsBlankLine = !previousWasComment && e.CommentMode != CommentsPrecede
			} else if previousWasComment {
				needsBlankLine = e.CommentMode != CommentsFollow
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
