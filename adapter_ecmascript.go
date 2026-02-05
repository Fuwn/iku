package main

import (
	"github.com/Fuwn/iku/engine"
	"strings"
)

type EcmaScriptAdapter struct{}

func (a *EcmaScriptAdapter) Analyze(source []byte) ([]byte, []engine.LineEvent, error) {
	sourceLines := strings.Split(string(source), "\n")
	events := make([]engine.LineEvent, len(sourceLines))
	insideTemplateString := false
	insideBlockComment := false
	previousEndedWithContinuation := false

	for lineIndex, currentLine := range sourceLines {
		backtickCount := countRawStringDelimiters(currentLine)
		wasInsideTemplateString := insideTemplateString

		if backtickCount%2 == 1 {
			insideTemplateString = !insideTemplateString
		}

		event := engine.NewLineEvent(currentLine)

		if wasInsideTemplateString {
			event.InRawString = true
			events[lineIndex] = event

			continue
		}

		if event.IsBlank {
			events[lineIndex] = event

			continue
		}

		trimmedContent := event.TrimmedContent

		if insideBlockComment {
			event.IsCommentOnly = true

			if strings.Contains(trimmedContent, "*/") {
				insideBlockComment = false
			}

			events[lineIndex] = event
			previousEndedWithContinuation = false

			continue
		}

		if strings.HasPrefix(trimmedContent, "/*") {
			event.IsCommentOnly = true

			if !strings.Contains(trimmedContent, "*/") {
				insideBlockComment = true
			}

			events[lineIndex] = event
			previousEndedWithContinuation = false

			continue
		}

		event.IsClosingBrace = isClosingBrace(currentLine)
		event.IsOpeningBrace = isOpeningBrace(currentLine)
		event.IsCaseLabel = isCaseLabel(currentLine)
		event.IsCommentOnly = isCommentOnly(currentLine)

		if event.IsCommentOnly {
			events[lineIndex] = event

			continue
		}

		isContinuationLine := previousEndedWithContinuation ||
			strings.HasPrefix(trimmedContent, ".") ||
			strings.HasPrefix(trimmedContent, "?.") ||
			strings.HasPrefix(trimmedContent, "]")
		previousEndedWithContinuation = ecmaScriptLineEndsContinuation(trimmedContent)

		if isClosingCurlyBrace(currentLine) {
			event.HasASTInfo = true
			event.IsScoped = true
			event.IsTopLevel = ecmaScriptLineIsTopLevel(currentLine)
			events[lineIndex] = event

			continue
		}

		if isContinuationLine {
			events[lineIndex] = event

			continue
		}

		statementType, isScoped := classifyEcmaScriptStatement(trimmedContent)

		if statementType != "" {
			event.HasASTInfo = true
			event.StatementType = statementType
			event.IsScoped = isScoped
			event.IsTopLevel = ecmaScriptLineIsTopLevel(currentLine)
			event.IsStartLine = true
		} else {
			event.HasASTInfo = true
			event.StatementType = "expression"
			event.IsTopLevel = ecmaScriptLineIsTopLevel(currentLine)
		}

		events[lineIndex] = event
	}

	return source, events, nil
}

func classifyEcmaScriptStatement(trimmedLine string) (string, bool) {
	classified := trimmedLine

	if strings.HasPrefix(classified, "export default ") {
		classified = classified[15:]
	} else if strings.HasPrefix(classified, "export ") {
		classified = classified[7:]
	}

	if strings.HasPrefix(classified, "async ") {
		classified = classified[6:]
	}

	if strings.HasPrefix(classified, "declare ") {
		classified = classified[8:]
	}

	switch {
	case ecmaScriptStatementHasPrefix(classified, "function"):
		return "function", true
	case ecmaScriptStatementHasPrefix(classified, "class"):
		return "class", true
	case ecmaScriptStatementHasPrefix(classified, "if"):
		return "if", true
	case ecmaScriptStatementHasPrefix(classified, "else"):
		return "if", true
	case ecmaScriptStatementHasPrefix(classified, "for"):
		return "for", true
	case ecmaScriptStatementHasPrefix(classified, "while"):
		return "while", true
	case ecmaScriptStatementHasPrefix(classified, "do"):
		return "do", true
	case ecmaScriptStatementHasPrefix(classified, "switch"):
		return "switch", true
	case ecmaScriptStatementHasPrefix(classified, "try"):
		return "try", true
	case ecmaScriptStatementHasPrefix(classified, "interface"):
		return "interface", true
	case ecmaScriptStatementHasPrefix(classified, "enum"):
		return "enum", true
	case ecmaScriptStatementHasPrefix(classified, "namespace"):
		return "namespace", true
	case ecmaScriptStatementHasPrefix(classified, "const"):
		return "const", false
	case ecmaScriptStatementHasPrefix(classified, "let"):
		return "let", false
	case ecmaScriptStatementHasPrefix(classified, "var"):
		return "var", false
	case ecmaScriptStatementHasPrefix(classified, "import"):
		return "import", false
	case ecmaScriptStatementHasPrefix(classified, "type"):
		return "type", false
	case ecmaScriptStatementHasPrefix(classified, "return"):
		return "return", false
	case ecmaScriptStatementHasPrefix(classified, "throw"):
		return "throw", false
	case ecmaScriptStatementHasPrefix(classified, "await"):
		return "await", false
	case ecmaScriptStatementHasPrefix(classified, "yield"):
		return "yield", false
	}

	return "", false
}

func ecmaScriptStatementHasPrefix(line string, keyword string) bool {
	if !strings.HasPrefix(line, keyword) {
		return false
	}

	if len(line) == len(keyword) {
		return true
	}

	nextCharacter := line[len(keyword)]

	return nextCharacter == ' ' || nextCharacter == '(' || nextCharacter == '{' ||
		nextCharacter == ';' || nextCharacter == '<' || nextCharacter == '\t'
}

func ecmaScriptLineIsTopLevel(sourceLine string) bool {
	if len(sourceLine) == 0 {
		return false
	}

	return !isWhitespace(sourceLine[0])
}

func ecmaScriptLineEndsContinuation(trimmedLine string) bool {
	if len(trimmedLine) == 0 {
		return false
	}

	lastCharacter := trimmedLine[len(trimmedLine)-1]

	if lastCharacter == ',' || lastCharacter == '[' || lastCharacter == '(' {
		return true
	}

	if lastCharacter == '>' && strings.HasPrefix(trimmedLine, "<") {
		return true
	}

	return false
}

func isClosingCurlyBrace(sourceLine string) bool {
	for characterIndex := 0; characterIndex < len(sourceLine); characterIndex++ {
		character := sourceLine[characterIndex]

		if isWhitespace(character) {
			continue
		}

		return character == '}'
	}

	return false
}
