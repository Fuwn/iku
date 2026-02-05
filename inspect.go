package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

func isGeneralDeclarationScoped(generalDeclaration *ast.GenDecl) bool {
	for _, specification := range generalDeclaration.Specs {
		if typeSpecification, isTypeSpecification := specification.(*ast.TypeSpec); isTypeSpecification {
			switch typeSpecification.Type.(type) {
			case *ast.StructType, *ast.InterfaceType:
				return true
			}
		}
	}

	return false
}

func (f *Formatter) buildLineInfo(tokenFileSet *token.FileSet, parsedFile *ast.File) map[int]*lineInformation {
	lineInformationMap := make(map[int]*lineInformation, 2*len(parsedFile.Decls))
	tokenFile := tokenFileSet.File(parsedFile.Pos())

	if tokenFile == nil {
		return lineInformationMap
	}

	for _, declaration := range parsedFile.Decls {
		startLine := tokenFile.Line(declaration.Pos())
		endLine := tokenFile.Line(declaration.End())
		statementType := ""
		isScoped := false

		switch typedDeclaration := declaration.(type) {
		case *ast.GenDecl:
			statementType = typedDeclaration.Tok.String()
			isScoped = isGeneralDeclarationScoped(typedDeclaration)
		case *ast.FuncDecl:
			statementType = "func"
			isScoped = true
		default:
			statementType = fmt.Sprintf("%T", declaration)
		}

		lineInformationMap[startLine] = &lineInformation{statementType: statementType, isTopLevel: true, isScoped: isScoped, isStartLine: true}
		lineInformationMap[endLine] = &lineInformation{statementType: statementType, isTopLevel: true, isScoped: isScoped, isStartLine: false}
	}

	ast.Inspect(parsedFile, func(astNode ast.Node) bool {
		if astNode == nil {
			return true
		}

		switch typedNode := astNode.(type) {
		case *ast.BlockStmt:
			f.processBlock(tokenFile, typedNode, lineInformationMap)
		case *ast.CaseClause:
			f.processStatementList(tokenFile, typedNode.Body, lineInformationMap)
		case *ast.CommClause:
			f.processStatementList(tokenFile, typedNode.Body, lineInformationMap)
		}

		return true
	})

	return lineInformationMap
}

func (f *Formatter) processBlock(tokenFile *token.File, blockStatement *ast.BlockStmt, lineInformationMap map[int]*lineInformation) {
	if blockStatement == nil {
		return
	}

	f.processStatementList(tokenFile, blockStatement.List, lineInformationMap)
}

func (f *Formatter) processStatementList(tokenFile *token.File, statements []ast.Stmt, lineInformationMap map[int]*lineInformation) {
	for _, statement := range statements {
		startLine := tokenFile.Line(statement.Pos())
		endLine := tokenFile.Line(statement.End())
		statementType := ""
		isScoped := false

		switch typedStatement := statement.(type) {
		case *ast.DeclStmt:
			if generalDeclaration, isGeneralDeclaration := typedStatement.Decl.(*ast.GenDecl); isGeneralDeclaration {
				statementType = generalDeclaration.Tok.String()
			} else {
				statementType = fmt.Sprintf("%T", statement)
			}
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
			*ast.TypeSwitchStmt, *ast.SelectStmt, *ast.BlockStmt:
			statementType = fmt.Sprintf("%T", statement)
			isScoped = true
		default:
			statementType = fmt.Sprintf("%T", statement)
		}

		existingStart := lineInformationMap[startLine]

		if existingStart == nil || !existingStart.isStartLine {
			lineInformationMap[startLine] = &lineInformation{statementType: statementType, isTopLevel: false, isScoped: isScoped, isStartLine: true}
		}

		existingEnd := lineInformationMap[endLine]

		if existingEnd == nil || !existingEnd.isStartLine {
			lineInformationMap[endLine] = &lineInformation{statementType: statementType, isTopLevel: false, isScoped: isScoped, isStartLine: false}
		}

		switch typedStatement := statement.(type) {
		case *ast.IfStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInformationMap)

			if typedStatement.Else != nil {
				if elseBlock, isBlockStatement := typedStatement.Else.(*ast.BlockStmt); isBlockStatement {
					f.processBlock(tokenFile, elseBlock, lineInformationMap)
				} else if elseIfStatement, isIfStatement := typedStatement.Else.(*ast.IfStmt); isIfStatement {
					f.processIfStatement(tokenFile, elseIfStatement, lineInformationMap)
				}
			}
		case *ast.ForStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInformationMap)
		case *ast.RangeStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInformationMap)
		case *ast.SwitchStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInformationMap)
		case *ast.TypeSwitchStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInformationMap)
		case *ast.SelectStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInformationMap)
		case *ast.BlockStmt:
			f.processBlock(tokenFile, typedStatement, lineInformationMap)
		}
	}
}

func (f *Formatter) processIfStatement(tokenFile *token.File, ifStatement *ast.IfStmt, lineInformationMap map[int]*lineInformation) {
	startLine := tokenFile.Line(ifStatement.Pos())
	endLine := tokenFile.Line(ifStatement.End())
	existingStart := lineInformationMap[startLine]

	if existingStart == nil || !existingStart.isStartLine {
		lineInformationMap[startLine] = &lineInformation{statementType: "*ast.IfStmt", isTopLevel: false, isScoped: true, isStartLine: true}
	}

	existingEnd := lineInformationMap[endLine]

	if existingEnd == nil || !existingEnd.isStartLine {
		lineInformationMap[endLine] = &lineInformation{statementType: "*ast.IfStmt", isTopLevel: false, isScoped: true, isStartLine: false}
	}

	f.processBlock(tokenFile, ifStatement.Body, lineInformationMap)

	if ifStatement.Else != nil {
		if elseBlock, isBlockStatement := ifStatement.Else.(*ast.BlockStmt); isBlockStatement {
			f.processBlock(tokenFile, elseBlock, lineInformationMap)
		} else if elseIfStatement, isIfStatement := ifStatement.Else.(*ast.IfStmt); isIfStatement {
			f.processIfStatement(tokenFile, elseIfStatement, lineInformationMap)
		}
	}
}
