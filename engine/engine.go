package engine

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

func (e *Engine) Format(events []LineEvent) []byte {
	outputCapacityBytes := 1

	for _, event := range events {
		if !event.IsBlank || event.InRawString {
			outputCapacityBytes += len(event.Content) + 2
		}
	}

	output := make([]byte, 0, outputCapacityBytes)
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

		needsBlankLine := false

		if !event.InRawString && hasWrittenContent && !previousWasOpenBrace && !event.IsClosingBrace && !event.IsCaseLabel && !event.IsContinuation {
			currentIsTopLevel := event.HasASTInfo && event.IsTopLevel
			currentIsScoped := event.HasASTInfo && event.IsScoped
			currentIsSingleLineScope := currentIsScoped && event.IsSingleLineScope
			sameStatementType := event.StatementType == previousStatementType
			groupedBindings := sameStatementType && event.StatementType == "binding" && !currentIsTopLevel && !previousWasTopLevel
			groupedCalls := sameStatementType && event.StatementType == "call"
			isDefinition := event.StatementType == "function" || event.StatementType == "type"

			if currentIsTopLevel && previousWasTopLevel && !sameStatementType {
				needsBlankLine = true
			} else if groupedBindings || groupedCalls {
				needsBlankLine = false
			} else if event.HasASTInfo && (currentIsScoped || previousWasScoped) {
				groupedDefinitions := e.GroupSingleLineScopes && isDefinition && currentIsSingleLineScope && previousWasSingleLineScope && sameStatementType

				needsBlankLine = !groupedDefinitions
			} else if event.StatementType != "" && previousStatementType != "" && !sameStatementType {
				needsBlankLine = true
			}

			if event.IsCommentOnly {
				needsBlankLine = !previousWasComment && e.CommentMode != CommentsPrecede
			} else if previousWasComment {
				needsBlankLine = e.CommentMode != CommentsFollow
			}
		}

		if needsBlankLine {
			output = append(output, '\n')
		}

		if hasWrittenContent {
			output = append(output, '\n')
		}

		output = append(output, event.Content...)
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
		} else if !event.InRawString && event.StatementType != "" {
			previousStatementType = event.StatementType
			previousWasTopLevel = false
			previousWasScoped = false
			previousWasSingleLineScope = false
		}
	}

	return append(output, '\n')
}
