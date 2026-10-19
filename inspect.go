package main

import (
	"github.com/Fuwn/iku/engine"
	"go/ast"
	"go/token"
	"reflect"
)

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

	if tokenFile == nil {
		return
	}

	for _, declaration := range parsedFile.Decls {
		startLine := tokenFile.PositionFor(declaration.Pos(), false).Line
		endLine := tokenFile.PositionFor(declaration.End(), false).Line
		isSingleLine := startLine == endLine
		statementType := ""
		isScoped := false

		switch typedDeclaration := declaration.(type) {
		case *ast.GenDecl:
			statementType, isScoped = classifyGoGeneralDeclaration(tokenFile, typedDeclaration, isSingleLine)
		case *ast.FuncDecl:
			statementType = "function"
			isScoped = true
		default:
			statementType = reflect.TypeOf(declaration).String()
		}

		events[startLine-1] = engine.LineEvent{HasASTInfo: true, StatementType: statementType, IsTopLevel: true, IsScoped: isScoped, IsSingleLineScope: isSingleLine, IsStartLine: true}

		if endLine != startLine {
			events[endLine-1] = engine.LineEvent{HasASTInfo: true, StatementType: statementType, IsTopLevel: true, IsScoped: isScoped}
		}
	}

	ast.Inspect(parsedFile, func(astNode ast.Node) bool {
		switch typedNode := astNode.(type) {
		case *ast.BlockStmt:
			indexGoStatementList(tokenFile, typedNode.List, events)
		case *ast.CaseClause:
			indexGoStatementList(tokenFile, typedNode.Body, events)
		case *ast.CommClause:
			indexGoStatementList(tokenFile, typedNode.Body, events)
		case *ast.IfStmt:
			if elseIfStatement, isElseIf := typedNode.Else.(*ast.IfStmt); isElseIf {
				indexGoStatementList(tokenFile, []ast.Stmt{elseIfStatement}, events)
			}
		}

		return true
	})
}

func indexGoStatementList(tokenFile *token.File, statements []ast.Stmt, events []engine.LineEvent) {
	for _, statement := range statements {
		startLine := tokenFile.PositionFor(statement.Pos(), false).Line
		endLine := tokenFile.PositionFor(statement.End(), false).Line
		isSingleLine := startLine == endLine
		statementType := ""
		isScoped := false

		switch typedStatement := statement.(type) {
		case *ast.DeclStmt:
			if generalDeclaration, isGeneralDeclaration := typedStatement.Decl.(*ast.GenDecl); isGeneralDeclaration {
				statementType, isScoped = classifyGoGeneralDeclaration(tokenFile, generalDeclaration, isSingleLine)
			}
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
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
			*ast.TypeSwitchStmt, *ast.SelectStmt, *ast.BlockStmt:
			isScoped = true
		}

		if statementType == "" {
			statementType = reflect.TypeOf(statement).String()
		}

		if !events[startLine-1].IsStartLine {
			events[startLine-1] = engine.LineEvent{HasASTInfo: true, StatementType: statementType, IsScoped: isScoped, IsSingleLineScope: isSingleLine, IsStartLine: true}
		}

		if !events[endLine-1].IsStartLine {
			events[endLine-1] = engine.LineEvent{HasASTInfo: true, StatementType: statementType, IsScoped: isScoped}
		}
	}
}
