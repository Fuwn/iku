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

func (e *Engine) FormatToString(events []LineEvent) string {
	var resultBuilder strings.Builder
	outputCapacityBytes := 1

	for _, event := range events {
		if !event.IsBlank || event.InRawString {
			outputCapacityBytes += len(event.Content) + 2
		}
	}

	resultBuilder.Grow(outputCapacityBytes)

	hasWrittenContent := false
	previousWasOpenBrace := false
	previousStatementType := ""
	previousWasComment := false
	previousWasTopLevel := false
	previousWasScoped := false
	previousWasSingleLineScope := false

	for _, event := range events {
		if event.IsBlank && !event.InRawString {
			continue
		}

		currentStatementType := event.StatementType

		if event.IsPackageDecl {
			currentStatementType = "package"
		}

		needsBlankLine := false

		if !event.InRawString && hasWrittenContent && !previousWasOpenBrace && !event.IsClosingBrace && !event.IsCaseLabel && !event.IsContinuation {
			currentIsTopLevel := event.HasASTInfo && event.IsTopLevel
			currentIsScoped := event.HasASTInfo && event.IsScoped
			currentIsSingleLineScope := currentIsScoped && event.IsSingleLineScope
			sameStatementType := currentStatementType == previousStatementType
			groupedBindings := sameStatementType && currentStatementType == "binding" && !currentIsTopLevel && !previousWasTopLevel
			groupedCalls := sameStatementType && currentStatementType == "call"
			isDefinition := currentStatementType == "function" || currentStatementType == "type"

			if currentIsTopLevel && previousWasTopLevel && !sameStatementType {
				needsBlankLine = true
			} else if groupedBindings || groupedCalls {
				needsBlankLine = false
			} else if event.HasASTInfo && (currentIsScoped || previousWasScoped) {
				groupedDefinitions := e.GroupSingleLineScopes && isDefinition && currentIsSingleLineScope && previousWasSingleLineScope && sameStatementType

				needsBlankLine = !groupedDefinitions
			} else if currentStatementType != "" && previousStatementType != "" && !sameStatementType {
				needsBlankLine = true
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

		if !event.InRawString || event.HasASTInfo {
			previousWasComment = event.IsCommentOnly
		}

		if event.HasASTInfo {
			previousStatementType = event.StatementType
			previousWasTopLevel = event.IsTopLevel
			previousWasScoped = event.IsScoped
			previousWasSingleLineScope = event.IsSingleLineScope && event.IsScoped
		} else if !event.InRawString && currentStatementType != "" {
			previousStatementType = currentStatementType
			previousWasTopLevel = false
			previousWasScoped = false
			previousWasSingleLineScope = false
		}
	}

	resultBuilder.WriteByte('\n')

	return resultBuilder.String()
}

func (e *Engine) FormatToBytes(events []LineEvent) []byte {
	return []byte(e.FormatToString(events))
}
