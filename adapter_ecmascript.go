package main

import (
	"github.com/Fuwn/iku/engine"
	"strings"
)

type EcmaScriptAdapter struct{}

func (a *EcmaScriptAdapter) Analyze(source []byte) ([]byte, []engine.LineEvent, error) {
	sourceLines := strings.Split(string(source), "\n")
	events := make([]engine.LineEvent, len(sourceLines))
	lexicalState := ecmaScriptLexicalState{}
	delimiters := []ecmaScriptDelimiter{}
	rootStatement := -1
	pendingBody := -1
	previousCode := ""
	previousEndedWithContinuation := false

	for lineIndex, currentLine := range sourceLines {
		wasInsideLiteral := lexicalState.quote != 0
		wasInsideComment := lexicalState.insideBlockComment
		code := strings.TrimSpace(lexicalState.maskLiteralsAndComments(currentLine))
		event := engine.NewLineEvent(currentLine)

		event.InRawString = wasInsideLiteral || wasInsideComment
		event.IsCommentOnly = code == "" && !event.IsBlank && !wasInsideLiteral

		if code == "" {
			events[lineIndex] = event

			continue
		}

		statementStart := rootStatement
		insideExpression := false

		if len(delimiters) > 0 {
			parent := delimiters[len(delimiters)-1]

			insideExpression = !parent.isBody
			statementStart = parent.currentStatement

			if insideExpression {
				statementStart = parent.statementStart
			}
		}

		if pendingBody >= 0 {
			statementStart = pendingBody
		}

		event.IsClosingBrace = isClosingBrace(code) || strings.HasPrefix(code, "]")
		event.IsOpeningBrace = isOpeningBrace(code) || strings.HasSuffix(code, "[")
		event.IsCaseLabel = isCaseLabel(code)

		continuedBody := strings.HasPrefix(code, "{") && statementStart >= 0 && events[statementStart].IsScoped
		isContinuationLine := wasInsideLiteral || insideExpression || previousEndedWithContinuation || continuedBody || pendingBody >= 0 ||
			strings.HasPrefix(code, ".") || strings.HasPrefix(code, "?.")

		if !isContinuationLine && !event.IsClosingBrace && !event.IsCaseLabel {
			statementType, isScoped, isContinuation := classifyEcmaScriptStatement(code)

			if statementType == "call" && len(delimiters) > 0 {
				parent := delimiters[len(delimiters)-1]

				if parent.isBody && parent.statementStart >= 0 && events[parent.statementStart].StatementType == "type" {
					statementType = "function"
					isScoped = true
				}
			}

			event.HasASTInfo = true
			event.StatementType = statementType
			event.IsScoped = isScoped
			event.IsSingleLineScope = isScoped
			event.IsContinuation = isContinuation
			event.IsTopLevel = len(delimiters) == 0
			event.IsStartLine = true
			statementStart = lineIndex

			if isScoped && (statementType == "function" || statementType == "type" || statementType == "namespace") && !strings.HasSuffix(code, ";") {
				pendingBody = lineIndex
			}

			if len(delimiters) == 0 {
				rootStatement = lineIndex
			} else {
				delimiters[len(delimiters)-1].currentStatement = lineIndex
			}
		}

		if statementStart >= 0 && ecmaScriptHasFunctionKeyword(code) && !strings.HasSuffix(code, ";") {
			pendingBody = statementStart
		}

		events[lineIndex] = event

		for characterIndex := 0; characterIndex < len(code); characterIndex++ {
			character := code[characterIndex]

			switch character {
			case '(', '[', '{':
				prefix := strings.TrimSpace(code[:characterIndex])
				continuedFunctionBody := prefix == "" && pendingBody == statementStart && strings.Contains(previousCode, ")")
				isBody := character == '{' && (continuedFunctionBody || ecmaScriptBraceIsBody(prefix, statementStart, events, delimiters))
				closing := byte(')')

				switch character {
				case '[':
					closing = ']'
				case '{':
					closing = '}'
				}

				if isBody && pendingBody == statementStart {
					pendingBody = -1
				}

				if isBody && statementStart == lineIndex && event.StatementType == "call" && strings.HasSuffix(prefix, ")") {
					events[lineIndex].StatementType = "function"
					events[lineIndex].IsScoped = true
				}

				delimiters = append(delimiters, ecmaScriptDelimiter{closing: closing, isBody: isBody, openingLine: lineIndex, statementStart: statementStart, currentStatement: -1})
			case ')', ']', '}':
				if len(delimiters) == 0 || delimiters[len(delimiters)-1].closing != character {
					continue
				}

				closed := delimiters[len(delimiters)-1]

				delimiters = delimiters[:len(delimiters)-1]

				if closed.statementStart < 0 {
					continue
				}

				owner := &events[closed.statementStart]

				if closed.isBody && closed.openingLine != lineIndex {
					owner.IsScoped = true
				}

				if closed.isBody {
					owner.IsSingleLineScope = closed.statementStart == lineIndex
				}

				if event.IsClosingBrace {
					events[lineIndex].HasASTInfo = true
					events[lineIndex].StatementType = owner.StatementType
					events[lineIndex].IsScoped = owner.IsScoped
					events[lineIndex].IsSingleLineScope = owner.IsSingleLineScope
					events[lineIndex].IsTopLevel = owner.IsTopLevel
					statementStart = closed.statementStart
				}
			}
		}

		previousEndedWithContinuation = ecmaScriptLineEndsContinuation(code)
		previousCode = code
	}

	return source, events, nil
}

func classifyEcmaScriptStatement(trimmedLine string) (string, bool, bool) {
	classified := trimmedLine

	if strings.HasPrefix(classified, "export default ") {
		classified = classified[15:]
	} else if strings.HasPrefix(classified, "export ") {
		classified = classified[7:]
	}

	classified = strings.TrimPrefix(classified, "async ")
	classified = strings.TrimPrefix(classified, "declare ")

	if classified == "" {
		return "", false, false
	}

	switch {
	case ecmaScriptStatementHasPrefix(classified, "function"):
		return "function", true, false
	case ecmaScriptStatementHasPrefix(classified, "class"):
		return "type", true, false
	case ecmaScriptStatementHasPrefix(classified, "if"):
		return "if", true, false
	case ecmaScriptStatementHasPrefix(classified, "else"):
		return "if", true, true
	case ecmaScriptStatementHasPrefix(classified, "for"):
		return "for", true, false
	case ecmaScriptStatementHasPrefix(classified, "while"):
		return "while", true, false
	case ecmaScriptStatementHasPrefix(classified, "do"):
		return "do", true, false
	case ecmaScriptStatementHasPrefix(classified, "switch"):
		return "switch", true, false
	case ecmaScriptStatementHasPrefix(classified, "try"):
		return "try", true, false
	case ecmaScriptStatementHasPrefix(classified, "catch"):
		return "try", true, true
	case ecmaScriptStatementHasPrefix(classified, "finally"):
		return "try", true, true
	case ecmaScriptStatementHasPrefix(classified, "interface"):
		return "type", true, false
	case ecmaScriptStatementHasPrefix(classified, "enum"):
		return "type", true, false
	case ecmaScriptStatementHasPrefix(classified, "namespace"):
		return "namespace", true, false
	case ecmaScriptStatementHasPrefix(classified, "const"):
		return "binding", false, false
	case ecmaScriptStatementHasPrefix(classified, "let"):
		return "binding", false, false
	case ecmaScriptStatementHasPrefix(classified, "var"):
		return "binding", false, false
	case ecmaScriptStatementHasPrefix(classified, "import"):
		return "import", false, false
	case ecmaScriptStatementHasPrefix(classified, "type"):
		return "type", false, false
	case ecmaScriptStatementHasPrefix(classified, "return"):
		return "return", false, false
	case ecmaScriptStatementHasPrefix(classified, "throw"):
		return "throw", false, false
	case ecmaScriptStatementHasPrefix(classified, "yield"):
		return "yield", false, false
	case ecmaScriptStatementHasPrefix(classified, "break"), ecmaScriptStatementHasPrefix(classified, "continue"):
		return "branch", false, false
	}

	if ecmaScriptHasAssignment(classified) {
		return "assignment", false, false
	}

	if strings.Contains(classified, "(") {
		return "call", false, false
	}

	return "expression", false, false
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

func ecmaScriptLineEndsContinuation(trimmedLine string) bool {
	if len(trimmedLine) == 0 {
		return false
	}

	lastCharacter := trimmedLine[len(trimmedLine)-1]

	if lastCharacter == ',' || lastCharacter == '[' || lastCharacter == '(' || lastCharacter == '=' ||
		strings.HasSuffix(trimmedLine, "=>") {
		return true
	}

	if lastCharacter == '>' && strings.HasPrefix(trimmedLine, "<") {
		return true
	}

	return false
}
