package main

import (
	"github.com/Fuwn/iku/engine"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

func analyzeGo(source []byte) ([]byte, []engine.LineEvent, error) {
	formattedSource, err := format.Source(source)

	if err != nil {
		return nil, nil, err
	}

	tokenFileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(tokenFileSet, "", formattedSource, parser.ParseComments|parser.SkipObjectResolution)

	if err != nil {
		return nil, nil, err
	}

	remainingSource := string(formattedSource)
	events := make([]engine.LineEvent, strings.Count(remainingSource, "\n")+1)

	indexGoStatements(tokenFileSet, parsedFile, events)

	events[tokenFileSet.PositionFor(parsedFile.Package, false).Line-1].StatementType = "package"

	for lineIndex := range events {
		currentLine, nextSource, _ := strings.Cut(remainingSource, "\n")
		event := &events[lineIndex]

		remainingSource = nextSource
		event.Content = currentLine
		event.IsBlank = strings.TrimSpace(currentLine) == ""
		event.IsClosingBrace = isClosingBrace(currentLine)
		event.IsOpeningBrace = isOpeningBrace(currentLine)
		event.IsCommentOnly = isCommentOnly(currentLine)
	}

	for _, commentGroup := range parsedFile.Comments {
		for _, comment := range commentGroup.List {
			if !strings.HasPrefix(comment.Text, "/*") {
				continue
			}

			firstLine := tokenFileSet.PositionFor(comment.Pos(), false).Line
			lastLine := tokenFileSet.PositionFor(comment.End(), false).Line

			for commentLine := firstLine; commentLine <= lastLine; commentLine++ {
				event := &events[commentLine-1]
				trimmedContent := strings.TrimSpace(event.Content)
				commentEnd := strings.Index(trimmedContent, "*/")
				commentOnly := strings.HasPrefix(trimmedContent, "/*") &&
					(commentEnd < 0 || strings.TrimSpace(trimmedContent[commentEnd+2:]) == "")

				event.IsVerbatim = commentLine != firstLine
				event.IsCommentOnly = event.IsCommentOnly || commentOnly
			}
		}
	}

	// Mark raw strings last because their closing lines can also start block comments.
	ast.Inspect(parsedFile, func(node ast.Node) bool {
		literal, isLiteral := node.(*ast.BasicLit)

		if !isLiteral || literal.Kind != token.STRING || !strings.HasPrefix(literal.Value, "`") {
			return true
		}

		firstLine := tokenFileSet.PositionFor(literal.Pos(), false).Line
		lastLine := tokenFileSet.PositionFor(literal.End(), false).Line

		for literalLine := firstLine + 1; literalLine <= lastLine; literalLine++ {
			events[literalLine-1].IsVerbatim = true
		}

		return false
	})

	return formattedSource, events, nil
}

func classifyGoGeneralDeclaration(tokenFile *token.File, generalDeclaration *ast.GenDecl, isSingleLine bool) (string, bool) {
	statementType := "binding"

	if generalDeclaration.Tok != token.VAR && generalDeclaration.Tok != token.CONST {
		statementType = generalDeclaration.Tok.String()
	}

	for _, specification := range generalDeclaration.Specs {
		if typeSpecification, isTypeSpecification := specification.(*ast.TypeSpec); isTypeSpecification {
			switch typeSpecification.Type.(type) {
			case *ast.StructType, *ast.InterfaceType:
				return statementType, true
			}
		}
	}

	return statementType, !isSingleLine && containsMultilineBody(tokenFile, generalDeclaration)
}

func containsMultilineBody(tokenFile *token.File, node ast.Node) bool {
	hasBody := false

	ast.Inspect(node, func(currentNode ast.Node) bool {
		if currentNode == nil || hasBody {
			return false
		}

		switch currentNode.(type) {
		case *ast.FuncLit, *ast.StructType, *ast.InterfaceType:
			hasBody = tokenFile.PositionFor(currentNode.Pos(), false).Line != tokenFile.PositionFor(currentNode.End(), false).Line
		}

		return !hasBody
	})

	return hasBody
}

func indexGoStatements(tokenFileSet *token.FileSet, parsedFile *ast.File, events []engine.LineEvent) {
	tokenFile := tokenFileSet.File(parsedFile.Pos())
	isStatementStart := make([]bool, len(events))

	for _, declaration := range parsedFile.Decls {
		startLine := tokenFile.PositionFor(declaration.Pos(), false).Line
		endLine := tokenFile.PositionFor(declaration.End(), false).Line
		isSingleLine := startLine == endLine
		statementType := "function"
		isScoped := true

		if generalDeclaration, isGeneralDeclaration := declaration.(*ast.GenDecl); isGeneralDeclaration {
			statementType, isScoped = classifyGoGeneralDeclaration(tokenFile, generalDeclaration, isSingleLine)
		}

		events[startLine-1] = engine.LineEvent{HasRole: true, StatementType: statementType, IsTopLevel: true, IsScoped: isScoped, IsSingleLine: isSingleLine}
		isStatementStart[startLine-1] = true

		if endLine != startLine {
			events[endLine-1] = engine.LineEvent{HasRole: true, StatementType: statementType, IsTopLevel: true, IsScoped: isScoped}
		}
	}

	ast.Inspect(parsedFile, func(astNode ast.Node) bool {
		switch typedNode := astNode.(type) {
		case *ast.BlockStmt:
			indexGoStatementList(tokenFile, typedNode.List, events, isStatementStart)
		case *ast.CaseClause:
			indexGoStatementList(tokenFile, typedNode.Body, events, isStatementStart)
			markGoCaseLabel(tokenFile, typedNode.Case, typedNode.Colon, events)
		case *ast.CommClause:
			indexGoStatementList(tokenFile, typedNode.Body, events, isStatementStart)
			markGoCaseLabel(tokenFile, typedNode.Case, typedNode.Colon, events)
		case *ast.IfStmt:
			if elseIfStatement, isElseIf := typedNode.Else.(*ast.IfStmt); isElseIf {
				indexGoStatementList(tokenFile, []ast.Stmt{elseIfStatement}, events, isStatementStart)
			}
		}

		return true
	})
}

// A case header can span lines; both its first line and its colon line border the clause body.
func markGoCaseLabel(tokenFile *token.File, casePosition, colonPosition token.Pos, events []engine.LineEvent) {
	events[tokenFile.PositionFor(casePosition, false).Line-1].IsCaseLabel = true
	events[tokenFile.PositionFor(colonPosition, false).Line-1].IsCaseLabel = true
}

func indexGoStatementList(tokenFile *token.File, statements []ast.Stmt, events []engine.LineEvent, isStatementStart []bool) {
	for _, statement := range statements {
		startLine := tokenFile.PositionFor(statement.Pos(), false).Line
		endLine := tokenFile.PositionFor(statement.End(), false).Line
		isSingleLine := startLine == endLine
		statementType := ""
		isScoped := false

		switch typedStatement := statement.(type) {
		case *ast.DeclStmt:
			statementType, isScoped = classifyGoGeneralDeclaration(tokenFile, typedStatement.Decl.(*ast.GenDecl), isSingleLine)
		case *ast.AssignStmt:
			statementType = "assignment"
			isScoped = !isSingleLine && containsMultilineBody(tokenFile, typedStatement)

			if typedStatement.Tok == token.DEFINE {
				statementType = "binding"
			}
		case *ast.ExprStmt:
			statementType = "expression"

			if _, isCall := typedStatement.X.(*ast.CallExpr); isCall {
				statementType = "call"
			}
		case *ast.IncDecStmt:
			statementType = "assignment"
		case *ast.ReturnStmt:
			statementType = "return"
		case *ast.DeferStmt:
			statementType = "defer"
		case *ast.GoStmt:
			statementType = "go"
		case *ast.SendStmt:
			statementType = "send"
		case *ast.BranchStmt:
			statementType = "branch"
		case *ast.LabeledStmt:
			statementType = "label"
		case *ast.CaseClause, *ast.CommClause:
			statementType = "case"
		case *ast.EmptyStmt:
			statementType = "empty"
		case *ast.IfStmt:
			statementType = "if"
			isScoped = true
		case *ast.ForStmt, *ast.RangeStmt:
			statementType = "for"
			isScoped = true
		case *ast.SwitchStmt, *ast.TypeSwitchStmt:
			statementType = "switch"
			isScoped = true
		case *ast.SelectStmt:
			statementType = "select"
			isScoped = true
		case *ast.BlockStmt:
			statementType = "block"
			isScoped = true
		}

		if !isStatementStart[startLine-1] {
			events[startLine-1] = engine.LineEvent{HasRole: true, StatementType: statementType, IsScoped: isScoped, IsSingleLine: isSingleLine}
			isStatementStart[startLine-1] = true
		}

		if !isStatementStart[endLine-1] {
			events[endLine-1] = engine.LineEvent{HasRole: true, StatementType: statementType, IsScoped: isScoped}
		}
	}
}

func isCommentOnly(sourceLine string) bool {
	return strings.HasPrefix(trimLeadingWhitespace(sourceLine), "//")
}
