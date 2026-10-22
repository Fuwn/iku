package main

import (
	"github.com/Fuwn/iku/engine"
	"strings"
)

type EcmaScriptAdapter struct{}

func (a *EcmaScriptAdapter) Analyze(source []byte) ([]byte, []engine.LineEvent, error) {
	remainingSource := string(source)
	events := make([]engine.LineEvent, strings.Count(remainingSource, "\n")+1)
	lexicalState := ecmaScriptLexicalState{}
	delimiters := []ecmaScriptDelimiter{}
	rootStatement := -1
	pendingBody := -1
	previousCode := ""
	previousEndedWithContinuation := false

	for lineIndex := range events {
		currentLine, nextSource, _ := strings.Cut(remainingSource, "\n")
		wasInsideLiteral := lexicalState.quote != 0
		wasInsideComment := lexicalState.insideBlockComment
		code := strings.TrimSpace(lexicalState.maskLiteralsAndComments(currentLine))
		event := engine.LineEvent{Content: currentLine, IsBlank: strings.TrimSpace(currentLine) == ""}

		remainingSource = nextSource
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
			statementStart = lineIndex

			if len(delimiters) == 0 {
				rootStatement = lineIndex
			} else {
				delimiters[len(delimiters)-1].currentStatement = lineIndex
			}
		}

		startsScopedDefinition := event.IsScoped &&
			(event.StatementType == "function" || event.StatementType == "type" || event.StatementType == "namespace")

		if (startsScopedDefinition || (statementStart >= 0 && ecmaScriptHasFunctionKeyword(code))) && !strings.HasSuffix(code, ";") {
			pendingBody = statementStart
		}

		events[lineIndex] = event

		for characterIndex := 0; characterIndex < len(code); characterIndex++ {
			character := code[characterIndex]

			switch character {
			case '(', '[', '{':
				closing := byte(')')
				isBody := false

				switch character {
				case '[':
					closing = ']'
				case '{':
					prefix := strings.TrimSpace(code[:characterIndex])
					continuedFunctionBody := prefix == "" && pendingBody == statementStart && strings.Contains(previousCode, ")")

					closing = '}'
					isBody = continuedFunctionBody || ecmaScriptBraceIsBody(prefix, statementStart, events, delimiters)

					if isBody && pendingBody == statementStart {
						pendingBody = -1
					}

					if isBody && statementStart == lineIndex && event.StatementType == "call" && strings.HasSuffix(prefix, ")") {
						events[lineIndex].StatementType = "function"
						events[lineIndex].IsScoped = true
					}
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
	classified, hasExportModifier := strings.CutPrefix(trimmedLine, "export ")

	if hasExportModifier {
		classified = strings.TrimPrefix(classified, "default ")
	}

	classified = strings.TrimPrefix(classified, "async ")
	classified = strings.TrimPrefix(classified, "declare ")

	if classified == "" {
		return "", false, false
	}

	keyword := classified

	if keywordEnd := strings.IndexAny(classified, " ({;<\t"); keywordEnd >= 0 {
		keyword = classified[:keywordEnd]
	}

	switch keyword {
	case "function":
		return "function", true, false
	case "class", "interface", "enum":
		return "type", true, false
	case "if", "for", "while", "do", "switch", "try", "namespace":
		return keyword, true, false
	case "else":
		return "if", true, true
	case "catch", "finally":
		return "try", true, true
	case "const", "let", "var":
		return "binding", false, false
	case "import", "type", "return", "throw", "yield":
		return keyword, false, false
	case "break", "continue":
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

func ecmaScriptLineEndsContinuation(trimmedLine string) bool {
	if len(trimmedLine) == 0 {
		return false
	}

	switch trimmedLine[len(trimmedLine)-1] {
	case ',', '[', '(', '=':
		return true
	case '>':
		return strings.HasSuffix(trimmedLine, "=>") || strings.HasPrefix(trimmedLine, "<")
	default:
		return false
	}
}
