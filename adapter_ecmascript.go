package main

import (
	"github.com/Fuwn/iku/engine"
	"strings"
)

type ecmaScriptDelimiter struct {
	closing          byte
	isBody           bool
	openingLine      int
	statementStart   int
	currentStatement int
}

type ecmaScriptAnalyzer struct {
	events                        []engine.LineEvent
	lexicalState                  ecmaScriptLexicalState
	delimiters                    []ecmaScriptDelimiter
	rootStatement                 int
	pendingBody                   int
	previousCode                  string
	previousEndedWithContinuation bool
}

func analyzeEcmaScript(source []byte) []engine.LineEvent {
	remainingSource := string(source)
	analyzer := ecmaScriptAnalyzer{
		events:        make([]engine.LineEvent, strings.Count(remainingSource, "\n")+1),
		rootStatement: -1,
		pendingBody:   -1,
	}

	for lineIndex := range analyzer.events {
		currentLine, nextSource, _ := strings.Cut(remainingSource, "\n")

		remainingSource = nextSource

		analyzer.analyzeLine(lineIndex, currentLine)
	}

	return analyzer.events
}

func (analyzer *ecmaScriptAnalyzer) analyzeLine(lineIndex int, currentLine string) {
	wasInsideLiteral := analyzer.lexicalState.quote != 0
	wasInsideComment := analyzer.lexicalState.insideBlockComment
	code := strings.TrimSpace(analyzer.lexicalState.maskLiteralsAndComments(currentLine))
	event := engine.LineEvent{Content: currentLine, IsBlank: strings.TrimSpace(currentLine) == ""}

	event.IsVerbatim = wasInsideLiteral || wasInsideComment
	event.IsCommentOnly = code == "" && !event.IsBlank && !wasInsideLiteral

	if code == "" {
		analyzer.events[lineIndex] = event

		return
	}

	statementStart, insideExpression := analyzer.enclosingStatement()

	event.IsClosingBrace = isClosingBrace(code) || strings.HasPrefix(code, "]") || strings.HasPrefix(code, ">")
	event.IsOpeningBrace = isOpeningBrace(code) || strings.HasSuffix(code, "[")
	event.IsCaseLabel = ecmaScriptIsCaseLabel(code)

	continuedBody := strings.HasPrefix(code, "{") && statementStart >= 0 && analyzer.events[statementStart].IsScoped
	isContinuationLine := wasInsideLiteral || insideExpression || analyzer.previousEndedWithContinuation || continuedBody ||
		analyzer.pendingBody >= 0 || ecmaScriptLineStartsContinuation(code)

	if !isContinuationLine && !event.IsClosingBrace && !event.IsCaseLabel {
		analyzer.startStatement(lineIndex, code, &event)

		statementStart = lineIndex
	}

	startsScopedDefinition := event.IsScoped &&
		(event.StatementType == "function" || event.StatementType == "type" || event.StatementType == "namespace")

	if (startsScopedDefinition || (statementStart >= 0 && ecmaScriptHasFunctionKeyword(code))) && !strings.HasSuffix(code, ";") {
		analyzer.pendingBody = statementStart
	}

	analyzer.events[lineIndex] = event

	analyzer.scanDelimiters(lineIndex, code, statementStart, event.StatementType, event.IsClosingBrace)

	analyzer.previousEndedWithContinuation = !event.IsCaseLabel && ecmaScriptLineEndsContinuation(code)
	analyzer.previousCode = code
}

// A line inside an expression delimiter belongs to the statement that opened it; inside a body
// it belongs to the body's latest statement. A pending function body claims the line first.
func (analyzer *ecmaScriptAnalyzer) enclosingStatement() (int, bool) {
	statementStart := analyzer.rootStatement
	insideExpression := false

	if len(analyzer.delimiters) > 0 {
		parent := analyzer.delimiters[len(analyzer.delimiters)-1]

		insideExpression = !parent.isBody
		statementStart = parent.currentStatement

		if insideExpression {
			statementStart = parent.statementStart
		}
	}

	if analyzer.pendingBody >= 0 {
		statementStart = analyzer.pendingBody
	}

	return statementStart, insideExpression
}

func (analyzer *ecmaScriptAnalyzer) startStatement(lineIndex int, code string, event *engine.LineEvent) {
	statementType, isScoped, isContinuation := classifyEcmaScriptStatement(code)

	event.HasRole = true
	event.StatementType = statementType
	event.IsScoped = isScoped
	event.IsSingleLine = true
	event.IsContinuation = isContinuation
	event.IsTopLevel = len(analyzer.delimiters) == 0

	if len(analyzer.delimiters) == 0 {
		analyzer.rootStatement = lineIndex

		return
	}

	parent := &analyzer.delimiters[len(analyzer.delimiters)-1]

	parent.currentStatement = lineIndex

	// Calls directly inside a class or interface body are method signatures.
	if statementType == "call" && parent.isBody && parent.statementStart >= 0 && analyzer.events[parent.statementStart].StatementType == "type" {
		promoteCallToMethod(event)
	}
}

func (analyzer *ecmaScriptAnalyzer) scanDelimiters(lineIndex int, code string, statementStart int, lineStatementType string, isClosingLine bool) {
	for characterIndex := 0; characterIndex < len(code); characterIndex++ {
		switch character := code[characterIndex]; character {
		case '(', '[':
			closing := byte(')')

			if character == '[' {
				closing = ']'
			}

			analyzer.delimiters = append(analyzer.delimiters, ecmaScriptDelimiter{closing: closing, openingLine: lineIndex, statementStart: statementStart, currentStatement: -1})
		case '{':
			prefix := strings.TrimSpace(code[:characterIndex])
			continuedFunctionBody := prefix == "" && analyzer.pendingBody == statementStart && strings.Contains(analyzer.previousCode, ")")
			isBody := continuedFunctionBody || ecmaScriptBraceIsBody(prefix, statementStart, analyzer.events, analyzer.delimiters)

			if isBody && analyzer.pendingBody == statementStart {
				analyzer.pendingBody = -1
			}

			// A call followed by a body on its own line is a method definition.
			if isBody && statementStart == lineIndex && lineStatementType == "call" && strings.HasSuffix(prefix, ")") {
				promoteCallToMethod(&analyzer.events[lineIndex])
			}

			analyzer.delimiters = append(analyzer.delimiters, ecmaScriptDelimiter{closing: '}', isBody: isBody, openingLine: lineIndex, statementStart: statementStart, currentStatement: -1})
		case ')', ']', '}':
			ownerStatement, closedOwnedDelimiter := analyzer.closeDelimiter(lineIndex, character)

			if closedOwnedDelimiter && isClosingLine {
				owner := analyzer.events[ownerStatement]
				closingEvent := &analyzer.events[lineIndex]

				closingEvent.HasRole = true
				closingEvent.StatementType = owner.StatementType
				closingEvent.IsScoped = owner.IsScoped
				closingEvent.IsSingleLine = owner.IsSingleLine
				closingEvent.IsTopLevel = owner.IsTopLevel
				statementStart = ownerStatement
			}
		}
	}
}

// Closing a body records whether its owner spans lines. Unmatched closers are ignored.
func (analyzer *ecmaScriptAnalyzer) closeDelimiter(lineIndex int, character byte) (int, bool) {
	if len(analyzer.delimiters) == 0 || analyzer.delimiters[len(analyzer.delimiters)-1].closing != character {
		return 0, false
	}

	closed := analyzer.delimiters[len(analyzer.delimiters)-1]

	analyzer.delimiters = analyzer.delimiters[:len(analyzer.delimiters)-1]

	if closed.statementStart < 0 {
		return 0, false
	}

	owner := &analyzer.events[closed.statementStart]

	if closed.isBody && closed.openingLine != lineIndex {
		owner.IsScoped = true
	}

	if closed.isBody {
		owner.IsSingleLine = closed.statementStart == lineIndex
	}

	return closed.statementStart, true
}

func promoteCallToMethod(event *engine.LineEvent) {
	event.StatementType = "function"
	event.IsScoped = true
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
	case ',', '[', '(', '<', '=', '?', ':', '&', '|', '^', '%', '*', '/':
		return true
	case '+', '-':
		return !strings.HasSuffix(trimmedLine, "++") && !strings.HasSuffix(trimmedLine, "--")
	case '>':
		return strings.HasSuffix(trimmedLine, "=>") || strings.HasPrefix(trimmedLine, "<")
	default:
		return false
	}
}

// Member access, ternary branches, and logical or type operators can lead a wrapped line.
func ecmaScriptLineStartsContinuation(trimmedLine string) bool {
	return trimmedLine != "" && strings.IndexByte(".?:|&", trimmedLine[0]) >= 0
}

func ecmaScriptHasFunctionKeyword(code string) bool {
	for searchStart := 0; searchStart < len(code); {
		relativeIndex := strings.Index(code[searchStart:], "function")

		if relativeIndex < 0 {
			return false
		}

		keywordStart := searchStart + relativeIndex
		keywordEnd := keywordStart + len("function")
		validStart := keywordStart == 0 || strings.ContainsRune(" \t=(:,;[{!", rune(code[keywordStart-1]))
		validEnd := keywordEnd == len(code) || strings.ContainsRune(" \t(*", rune(code[keywordEnd]))

		if validStart && validEnd {
			return true
		}

		searchStart = keywordEnd
	}

	return false
}

func ecmaScriptBraceIsBody(prefix string, statementStart int, events []engine.LineEvent, delimiters []ecmaScriptDelimiter) bool {
	if strings.HasSuffix(prefix, ")") || strings.HasSuffix(prefix, "=>") {
		return true
	}

	for _, keyword := range []string{"else", "try", "finally", "do"} {
		if prefix == keyword || strings.HasSuffix(prefix, " "+keyword) {
			return true
		}
	}

	if statementStart < 0 || (len(delimiters) > 0 && !delimiters[len(delimiters)-1].isBody) {
		return false
	}

	switch events[statementStart].StatementType {
	case "function", "type", "namespace":
		return true
	default:
		return prefix == ""
	}
}

func ecmaScriptHasAssignment(code string) bool {
	parenthesisDepth := 0
	bracketDepth := 0

	for characterIndex := 0; characterIndex < len(code); characterIndex++ {
		character := code[characterIndex]

		switch character {
		case '(':
			parenthesisDepth++
		case ')':
			parenthesisDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case '=':
			if parenthesisDepth != 0 || bracketDepth != 0 {
				continue
			}

			previousCharacter := byte(0)
			nextCharacter := byte(0)

			if characterIndex > 0 {
				previousCharacter = code[characterIndex-1]
			}

			if characterIndex+1 < len(code) {
				nextCharacter = code[characterIndex+1]
			}

			isComparison := previousCharacter == '=' || previousCharacter == '!' ||
				((previousCharacter == '<' || previousCharacter == '>') && (characterIndex < 2 || code[characterIndex-2] != previousCharacter))

			if !isComparison && nextCharacter != '=' && nextCharacter != '>' {
				return true
			}
		case '+', '-':
			if parenthesisDepth == 0 && bracketDepth == 0 && characterIndex+1 < len(code) && code[characterIndex+1] == character {
				return true
			}
		}
	}

	return false
}

func ecmaScriptIsCaseLabel(code string) bool {
	if afterCase, isCase := strings.CutPrefix(code, "case"); isCase && afterCase != "" && isWhitespace(afterCase[0]) {
		return true
	}

	afterDefault, isDefault := strings.CutPrefix(code, "default")

	return isDefault && strings.HasPrefix(trimLeadingWhitespace(afterDefault), ":")
}
