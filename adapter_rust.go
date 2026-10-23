package main

import (
	"github.com/Fuwn/iku/engine"
	"strings"
)

type rustDelimiterKind int

const (
	rustExpression rustDelimiterKind = iota
	rustBlock
	rustItemScope
	rustFieldList
	rustMatchArms
	rustMacroBody
)

// rustScanState follows the tokens written directly inside one delimiter, so a brace can be told
// apart as a body, a struct literal, a pattern, or a macro body.
type rustScanState struct {
	lastToken              string
	lastTokenIsWord        bool
	followsMacroBang       bool
	expectsClosureBody     bool
	expectsMacroDefinition bool
	pendingHeader          rustDelimiterKind
	awaitedPatternEnd      string
	onlyModifiers          bool
}

type rustDelimiter struct {
	closing                    byte
	kind                       rustDelimiterKind
	openingLine                int
	isMacroArguments           bool
	statementStart             int
	currentStatement           int
	currentStatementTerminated bool
	scan                       rustScanState
}

type rustAnalyzer struct {
	events []engine.LineEvent

	// Closing lines take their owner's final role once analysis ends. Each entry holds the
	// owner's line index plus one, so zero marks a line without an owner.
	closingLineOwners             []int
	lexicalState                  rustLexicalState
	delimiters                    []rustDelimiter
	macroBodyDepth                int
	attributedStatement           int
	previousEndedWithContinuation bool
}

func analyzeRust(source []byte) []engine.LineEvent {
	remainingSource := string(source)
	lineCount := strings.Count(remainingSource, "\n") + 1
	analyzer := rustAnalyzer{
		events:              make([]engine.LineEvent, lineCount),
		closingLineOwners:   make([]int, lineCount),
		delimiters:          []rustDelimiter{{kind: rustItemScope, statementStart: -1, currentStatement: -1, scan: newRustScanState()}},
		attributedStatement: -1,
	}

	for lineIndex := range analyzer.events {
		currentLine, nextSource, _ := strings.Cut(remainingSource, "\n")

		remainingSource = nextSource

		analyzer.analyzeLine(lineIndex, currentLine)
	}

	for lineIndex, ownerLinePlusOne := range analyzer.closingLineOwners {
		if ownerLinePlusOne == 0 {
			continue
		}

		owner := analyzer.events[ownerLinePlusOne-1]
		closingEvent := &analyzer.events[lineIndex]

		closingEvent.HasRole = true
		closingEvent.StatementType = owner.StatementType
		closingEvent.IsScoped = owner.IsScoped
		closingEvent.IsSingleLine = owner.IsSingleLine
		closingEvent.IsTopLevel = owner.IsTopLevel
	}

	return analyzer.events
}

func newRustScanState() rustScanState {
	return rustScanState{onlyModifiers: true}
}

func (analyzer *rustAnalyzer) analyzeLine(lineIndex int, currentLine string) {
	wasInsideLiteral := analyzer.lexicalState.isInsideLiteral()
	wasInsideComment := analyzer.lexicalState.blockCommentDepth > 0
	code := strings.TrimSpace(analyzer.lexicalState.maskLiteralsAndComments(currentLine))
	event := engine.LineEvent{Content: currentLine, IsBlank: strings.TrimSpace(currentLine) == ""}
	parent := &analyzer.delimiters[len(analyzer.delimiters)-1]

	event.IsVerbatim = wasInsideLiteral || wasInsideComment || analyzer.macroBodyDepth > 0
	event.IsCommentOnly = code == "" && !event.IsBlank && !wasInsideLiteral

	// Inner doc comments document their enclosing item, as the inner attributes they stand for do.
	if event.IsCommentOnly && !event.IsVerbatim && isRustInnerDocComment(currentLine) {
		event.IsCommentOnly = false
		event.HasRole = true
		event.StatementType = "attribute"
		event.IsSingleLine = true
		event.IsTopLevel = parent.kind == rustItemScope
	}

	// Fields and variants form a list, so their documentation stays packed with them.
	event.IsContinuation = parent.kind == rustFieldList

	if code == "" {
		analyzer.events[lineIndex] = event

		return
	}

	statementStart := parent.currentStatement

	if parent.kind == rustExpression {
		statementStart = parent.statementStart
	}

	event.IsClosingBrace = strings.IndexByte("})]", code[0]) >= 0
	event.IsOpeningBrace = strings.IndexByte("{([", code[len(code)-1]) >= 0

	isContinuationLine := wasInsideLiteral || parent.kind == rustExpression || analyzer.previousEndedWithContinuation ||
		parent.scan.pendingHeader != rustExpression || rustLineStartsContinuation(code)

	switch {
	case analyzer.macroBodyDepth > 0 || event.IsClosingBrace:
	case isContinuationLine:
		if analyzer.attributedStatement >= 0 && analyzer.attributedStatement == statementStart && code[0] != '#' {
			analyzer.classifyAttributedStatement(lineIndex, code)
		} else if parent.kind != rustExpression && statementStart >= 0 && code[0] == '.' {
			analyzer.classifyMethodChain(statementStart, code)
		}
	case parent.kind == rustFieldList:
	default:
		analyzer.startStatement(lineIndex, code, &event)

		statementStart = lineIndex
	}

	analyzer.events[lineIndex] = event

	analyzer.scanTokens(lineIndex, code, statementStart, event.IsClosingBrace)

	analyzer.previousEndedWithContinuation = analyzer.lineEndsContinuation(code)
}

func (analyzer *rustAnalyzer) startStatement(lineIndex int, code string, event *engine.LineEvent) {
	parent := &analyzer.delimiters[len(analyzer.delimiters)-1]

	event.HasRole = true
	event.IsSingleLine = true
	event.IsTopLevel = parent.kind == rustItemScope
	parent.currentStatement = lineIndex
	parent.currentStatementTerminated = false
	parent.scan = newRustScanState()
	analyzer.attributedStatement = -1

	// An arm with a multiline block is scoped like any other body; single-line arms group.
	if parent.kind == rustMatchArms {
		event.StatementType = "arm"

		return
	}

	statementType, isScoped, awaitsItem := classifyRustStatement(code)

	if awaitsItem {
		analyzer.attributedStatement = lineIndex

		return
	}

	event.StatementType = statementType
	event.IsScoped = isScoped
}

// An attribute's statement takes its role from the item below it.
func (analyzer *rustAnalyzer) classifyAttributedStatement(lineIndex int, code string) {
	statementType, isScoped, awaitsItem := classifyRustStatement(code)

	if awaitsItem {
		return
	}

	attributedEvent := &analyzer.events[analyzer.attributedStatement]

	attributedEvent.StatementType = statementType
	attributedEvent.IsScoped = isScoped
	attributedEvent.IsSingleLine = false
	analyzer.attributedStatement = -1
}

// A statement whose first line names only a receiver takes its role from the method chain
// wrapped below it.
func (analyzer *rustAnalyzer) classifyMethodChain(statementStart int, code string) {
	statement := &analyzer.events[statementStart]

	if statement.StatementType != "expression" {
		return
	}

	if chainType, _, _ := classifyRustStatement(code); chainType == "assignment" || chainType == "call" {
		statement.StatementType = chainType
	}
}

func (analyzer *rustAnalyzer) scanTokens(lineIndex int, code string, statementStart int, isClosingLine bool) {
	for tokenStart := 0; tokenStart < len(code); {
		character := code[tokenStart]

		if isWhitespace(character) {
			tokenStart++

			continue
		}

		if isRustIdentifierCharacter(character) || character == '\'' {
			tokenEnd := tokenStart + 1

			for tokenEnd < len(code) && isRustIdentifierCharacter(code[tokenEnd]) {
				tokenEnd++
			}

			analyzer.scanWord(code[tokenStart:tokenEnd], code[tokenEnd:])

			tokenStart = tokenEnd

			continue
		}

		switch character {
		case '(', '[', '{':
			analyzer.openDelimiter(lineIndex, character, statementStart)
		case ')', ']', '}':
			ownerStatement, closedOwnedDelimiter := analyzer.closeDelimiter(lineIndex, character)

			if closedOwnedDelimiter && isClosingLine {
				analyzer.closingLineOwners[lineIndex] = ownerStatement + 1
				statementStart = ownerStatement
			}
		default:
			punctuation := rustPunctuation(code[tokenStart:])

			analyzer.scanPunctuation(punctuation)

			tokenStart += len(punctuation)

			continue
		}

		tokenStart++
	}
}

func (analyzer *rustAnalyzer) scanWord(word string, followingCode string) {
	scan := &analyzer.delimiters[len(analyzer.delimiters)-1].scan

	followingCode = trimLeadingWhitespace(followingCode)

	isModifier := false

	switch word {
	case "if", "while", "match", "for":
		if scan.pendingHeader == rustExpression && (word != "for" || !strings.HasPrefix(followingCode, "<")) {
			scan.pendingHeader = rustBlock

			switch word {
			case "match":
				scan.pendingHeader = rustMatchArms
			case "for":
				scan.awaitedPatternEnd = "in"
			}
		}
	case "let":
		if scan.pendingHeader != rustExpression {
			scan.awaitedPatternEnd = "="
		}
	case "in":
		if scan.awaitedPatternEnd == "in" {
			scan.awaitedPatternEnd = ""
		}
	case "fn", "struct", "enum", "union", "trait", "impl", "mod", "extern":
		isItem := scan.onlyModifiers && (word != "union" || (followingCode != "" && isRustIdentifierCharacter(followingCode[0])))

		if isItem {
			scan.pendingHeader = rustItemKeywordBody(word)
		}

		isModifier = isItem && word == "extern"
	case "pub", "unsafe", "async", "const", "default":
		isModifier = true
	case "0":
		// The lexer masks an extern ABI string to this placeholder.
		isModifier = scan.lastToken == "extern"
	}

	scan.onlyModifiers = scan.onlyModifiers && isModifier
	scan.lastToken = word
	scan.lastTokenIsWord = true
	scan.followsMacroBang = false
}

func (analyzer *rustAnalyzer) scanPunctuation(punctuation string) {
	parent := &analyzer.delimiters[len(analyzer.delimiters)-1]
	scan := &parent.scan
	followsMacroBang := false

	// Commas also separate generic arguments and where-clause bounds, which cannot end a header.
	switch punctuation {
	case ";", ",":
		if punctuation == ";" || scan.pendingHeader == rustExpression {
			*scan = newRustScanState()
			parent.currentStatementTerminated = parent.currentStatementTerminated || punctuation == ";"

			return
		}
	case "=>":
		scan.pendingHeader = rustExpression
		scan.awaitedPatternEnd = ""
	case "->":
		scan.expectsClosureBody = scan.lastToken == "|" || scan.lastToken == "||"
	case "=":
		if scan.awaitedPatternEnd == "=" {
			scan.awaitedPatternEnd = ""
		}
	case "!":
		followsMacroBang = scan.lastTokenIsWord && !isRustKeyword(scan.lastToken)
		scan.expectsMacroDefinition = scan.lastToken == "macro_rules"
	}

	scan.onlyModifiers = scan.onlyModifiers && (punctuation == "#" || punctuation == "!")
	scan.lastToken = punctuation
	scan.lastTokenIsWord = false
	scan.followsMacroBang = followsMacroBang
}

func (analyzer *rustAnalyzer) openDelimiter(lineIndex int, character byte, statementStart int) {
	parent := &analyzer.delimiters[len(analyzer.delimiters)-1]
	scan := &parent.scan
	kind := rustExpression
	closing := byte(')')

	switch character {
	case '[':
		closing = ']'
	case '{':
		closing = '}'
		kind = scan.braceKind(parent.isMacroArguments)
	}

	// Struct-like and tuple variants list their own fields.
	if parent.kind == rustFieldList {
		kind = rustFieldList
	}

	if scan.expectsMacroDefinition {
		kind = rustMacroBody
	}

	if kind == rustMacroBody {
		analyzer.macroBodyDepth++
	}

	if kind != rustExpression {
		scan.pendingHeader = rustExpression
		scan.awaitedPatternEnd = ""

		if statementStart >= 0 && isRustDefinition(analyzer.events[statementStart].StatementType) {
			analyzer.events[statementStart].IsScoped = true
		}
	}

	isMacroArguments := kind == rustExpression && (parent.isMacroArguments || scan.followsMacroBang)

	scan.expectsMacroDefinition = false
	scan.followsMacroBang = false
	scan.expectsClosureBody = false
	analyzer.delimiters = append(analyzer.delimiters, rustDelimiter{
		closing:          closing,
		kind:             kind,
		openingLine:      lineIndex,
		isMacroArguments: isMacroArguments,
		statementStart:   statementStart,
		currentStatement: -1,
		scan:             newRustScanState(),
	})
}

// Closing a body records whether its owner spans lines. Unmatched closers are ignored.
func (analyzer *rustAnalyzer) closeDelimiter(lineIndex int, character byte) (int, bool) {
	if len(analyzer.delimiters) == 1 || analyzer.delimiters[len(analyzer.delimiters)-1].closing != character {
		return 0, false
	}

	closed := analyzer.delimiters[len(analyzer.delimiters)-1]

	analyzer.delimiters = analyzer.delimiters[:len(analyzer.delimiters)-1]

	parent := &analyzer.delimiters[len(analyzer.delimiters)-1]

	if closed.kind == rustMacroBody {
		analyzer.macroBodyDepth--
	}

	analyzer.markTailExpression(closed)

	if closed.kind == rustExpression {
		parent.scan.lastToken = string(character)
		parent.scan.lastTokenIsWord = false
		parent.scan.onlyModifiers = parent.scan.onlyModifiers && character != '}'
	} else {
		parent.scan = newRustScanState()
		parent.scan.lastToken = "}"
	}

	if closed.statementStart < 0 {
		return 0, false
	}

	owner := &analyzer.events[closed.statementStart]

	if closed.kind != rustExpression && closed.openingLine != lineIndex {
		owner.IsScoped = true
	}

	if closed.kind != rustExpression {
		owner.IsSingleLine = owner.IsSingleLine && closed.statementStart == lineIndex
	}

	return closed.statementStart, true
}

// A block's final unterminated call or expression is its value, so it reads as a return.
func (analyzer *rustAnalyzer) markTailExpression(closed rustDelimiter) {
	if closed.kind != rustBlock || closed.currentStatement < 0 || closed.currentStatementTerminated {
		return
	}

	if tail := &analyzer.events[closed.currentStatement]; tail.StatementType == "call" || tail.StatementType == "expression" {
		tail.StatementType = "return"
	}
}

// A brace opens a body after a pending item or control-flow header, a closure, a match arm, or a
// block keyword. After a path it opens a struct literal or pattern, as it does inside macro
// arguments, whose token trees need not be Rust expressions. Anywhere else it opens a block.
func (scan *rustScanState) braceKind(insideMacroArguments bool) rustDelimiterKind {
	if scan.followsMacroBang {
		return rustMacroBody
	}

	if scan.pendingHeader != rustExpression {
		if scan.awaitedPatternEnd != "" {
			return rustExpression
		}

		return scan.pendingHeader
	}

	if scan.expectsClosureBody {
		return rustBlock
	}

	switch scan.lastToken {
	case "=>", "|", "||", "else", "loop", "unsafe", "async", "move", "const":
		return rustBlock
	case ">", "::":
		return rustExpression
	}

	if insideMacroArguments || (scan.lastTokenIsWord && !isRustKeyword(scan.lastToken)) {
		return rustExpression
	}

	return rustBlock
}

func (analyzer *rustAnalyzer) lineEndsContinuation(code string) bool {
	parent := analyzer.delimiters[len(analyzer.delimiters)-1]

	if parent.kind != rustExpression && parent.currentStatement >= 0 && analyzer.events[parent.currentStatement].StatementType == "attribute" {
		return false
	}

	return strings.IndexByte(";{},", code[len(code)-1]) < 0
}

func isRustInnerDocComment(sourceLine string) bool {
	comment := trimLeadingWhitespace(sourceLine)

	return strings.HasPrefix(comment, "//!") || strings.HasPrefix(comment, "/*!")
}

func rustItemKeywordBody(keyword string) rustDelimiterKind {
	switch keyword {
	case "fn":
		return rustBlock
	case "struct", "enum", "union":
		return rustFieldList
	default:
		return rustItemScope
	}
}

func isRustDefinition(statementType string) bool {
	return statementType == "function" || statementType == "type" || statementType == "module" || statementType == "macro"
}

func isRustKeyword(word string) bool {
	switch word {
	case "if", "else", "while", "for", "loop", "match", "return", "break", "continue", "in", "let", "move", "mut", "ref", "as", "where":
		return true
	default:
		return false
	}
}

// Only operators that change how a brace or bang reads are joined; a closing generic bracket
// stays a single token so that `Foo<Bar<T>> {` still reads as a struct literal.
func rustPunctuation(code string) string {
	if len(code) >= 2 {
		switch code[:2] {
		case "=>", "->", "::", "||", "==", "!=", "<=", ">=":
			return code[:2]
		}
	}

	return code[:1]
}

// Method calls, error propagation, and binary operators can lead a wrapped line, as can return
// types and where clauses of a wrapped signature.
func rustLineStartsContinuation(code string) bool {
	switch code[0] {
	case '.', '?', '|', '+', '-', '/', '%', '^', '=', '>':
		return true
	case '&':
		return strings.HasPrefix(code, "&&")
	}

	return rustStartsWithWord(code, "as") || rustStartsWithWord(code, "where")
}

func rustStartsWithWord(code string, word string) bool {
	afterWord, hasWord := strings.CutPrefix(code, word)

	return hasWord && (afterWord == "" || !isRustIdentifierCharacter(afterWord[0]))
}

// classifyRustStatement returns the statement type and scope of a statement's first line, or
// reports that the line holds only outer attributes and the item below it decides the role.
func classifyRustStatement(code string) (string, bool, bool) {
	if strings.HasPrefix(code, "#!") {
		return "attribute", false, false
	}

	for strings.HasPrefix(code, "#[") {
		attributeEnd := rustClosingBracketIndex(code)

		if attributeEnd < 0 {
			return "", false, true
		}

		code = trimLeadingWhitespace(code[attributeEnd+1:])
	}

	if code == "" {
		return "", false, true
	}

	code = stripRustModifiers(stripRustLabel(code))

	keyword := rustLeadingWord(code)

	switch keyword {
	case "fn":
		return "function", false, false
	case "struct", "enum", "trait", "impl":
		return "type", true, false
	case "type":
		return "type", false, false
	case "union":
		if afterKeyword := trimLeadingWhitespace(code[len(keyword):]); afterKeyword != "" && isRustIdentifierCharacter(afterKeyword[0]) {
			return "type", true, false
		}
	case "mod":
		return "module", false, false
	case "use":
		return "use", false, false
	case "extern":
		if rustStartsWithWord(trimLeadingWhitespace(code[len(keyword):]), "crate") {
			return "crate", false, false
		}

		return "module", false, false
	case "let", "const", "static":
		return "binding", false, false
	case "if", "while", "for", "loop", "match":
		return keyword, true, false
	case "unsafe":
		if strings.HasPrefix(trimLeadingWhitespace(code[len(keyword):]), "{") {
			return "block", true, false
		}
	case "return":
		return "return", false, false
	case "break", "continue":
		return "branch", false, false
	case "macro_rules":
		return "macro", false, false
	case "":
		if strings.HasPrefix(code, "{") {
			return "block", true, false
		}
	}

	if rustHasAssignment(code) {
		return "assignment", false, false
	}

	if strings.Contains(code, "(") || strings.HasPrefix(code[len(keyword):], "!") {
		return "call", false, false
	}

	return "expression", false, false
}

func stripRustLabel(code string) string {
	if !strings.HasPrefix(code, "'") {
		return code
	}

	labelEnd := 1

	for labelEnd < len(code) && isRustIdentifierCharacter(code[labelEnd]) {
		labelEnd++
	}

	if afterLabel, isLabel := strings.CutPrefix(trimLeadingWhitespace(code[labelEnd:]), ":"); isLabel {
		return trimLeadingWhitespace(afterLabel)
	}

	return code
}

func stripRustModifiers(code string) string {
	for {
		keyword := rustLeadingWord(code)
		afterKeyword := trimLeadingWhitespace(code[len(keyword):])

		switch keyword {
		case "pub":
			if strings.HasPrefix(afterKeyword, "(") {
				if visibilityEnd := strings.IndexByte(afterKeyword, ')'); visibilityEnd >= 0 {
					afterKeyword = trimLeadingWhitespace(afterKeyword[visibilityEnd+1:])
				}
			}
		case "async", "unsafe", "default":
			if !isRustItemKeyword(rustLeadingWord(afterKeyword)) {
				return code
			}
		case "const":
			if followingKeyword := rustLeadingWord(afterKeyword); followingKeyword != "fn" && !isRustItemKeyword(followingKeyword) {
				return code
			}
		case "extern":
			afterAbi := afterKeyword

			if rustLeadingWord(afterKeyword) == "0" {
				afterAbi = trimLeadingWhitespace(afterKeyword[1:])
			}

			if !isRustItemKeyword(rustLeadingWord(afterAbi)) {
				return code
			}

			afterKeyword = afterAbi
		default:
			return code
		}

		code = afterKeyword
	}
}

func isRustItemKeyword(word string) bool {
	switch word {
	case "fn", "unsafe", "async", "const", "extern", "impl", "trait", "struct", "enum", "union", "type", "mod", "static", "use":
		return true
	default:
		return false
	}
}

func rustLeadingWord(code string) string {
	wordEnd := 0

	for wordEnd < len(code) && isRustIdentifierCharacter(code[wordEnd]) {
		wordEnd++
	}

	return code[:wordEnd]
}

// rustClosingBracketIndex returns the index of the bracket closing an attribute that opens the
// code, or -1 when the attribute continues past the line.
func rustClosingBracketIndex(code string) int {
	depth := 0

	for characterIndex := 0; characterIndex < len(code); characterIndex++ {
		switch code[characterIndex] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--

			if depth == 0 {
				return characterIndex
			}
		}
	}

	return -1
}

func rustHasAssignment(code string) bool {
	depth := 0

	for characterIndex := 0; characterIndex < len(code); characterIndex++ {
		switch code[characterIndex] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case '=':
			if depth != 0 {
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

			if nextCharacter == '=' || nextCharacter == '>' {
				characterIndex++

				continue
			}

			isComparison := previousCharacter == '!' || previousCharacter == '.' ||
				((previousCharacter == '<' || previousCharacter == '>') && (characterIndex < 2 || code[characterIndex-2] != previousCharacter))

			if !isComparison {
				return true
			}
		}
	}

	return false
}
