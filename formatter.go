package main

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

type CommentMode int

const (
	CommentsFollow CommentMode = iota
	CommentsPrecede
	CommentsStandalone
)

type Formatter struct {
	CommentMode CommentMode
}

type lineInformation struct {
	statementType string
	isTopLevel    bool
	isScoped      bool
	isStartLine   bool
}

func (f *Formatter) Format(source []byte) ([]byte, error) {
	formattedSource, err := format.Source(source)

	if err != nil {
		return nil, err
	}

	tokenFileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(tokenFileSet, "", formattedSource, parser.ParseComments)

	if err != nil {
		return nil, err
	}

	lineInformationMap := f.buildLineInfo(tokenFileSet, parsedFile)

	return f.rewrite(formattedSource, lineInformationMap), nil
}

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
	lineInformationMap := make(map[int]*lineInformation)
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
			statementType = reflect.TypeOf(declaration).String()
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
				statementType = reflect.TypeOf(statement).String()
			}
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
			*ast.TypeSwitchStmt, *ast.SelectStmt, *ast.BlockStmt:
			statementType = reflect.TypeOf(statement).String()
			isScoped = true
		default:
			statementType = reflect.TypeOf(statement).String()
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

func (f *Formatter) rewrite(formattedSource []byte, lineInformationMap map[int]*lineInformation) []byte {
	sourceLines := strings.Split(string(formattedSource), "\n")
	resultLines := make([]string, 0, len(sourceLines))
	previousWasOpenBrace := false
	previousStatementType := ""
	previousWasComment := false
	previousWasTopLevel := false
	previousWasScoped := false
	insideRawString := false

	for lineIndex, currentLine := range sourceLines {
		backtickCount := countRawStringDelimiters(currentLine)
		wasInsideRawString := insideRawString

		if backtickCount%2 == 1 {
			insideRawString = !insideRawString
		}

		if wasInsideRawString {
			resultLines = append(resultLines, currentLine)

			continue
		}

		lineNumber := lineIndex + 1
		trimmedLine := strings.TrimSpace(currentLine)

		if trimmedLine == "" {
			continue
		}

		isClosingBrace := closingBracePattern.MatchString(currentLine)
		isOpeningBrace := openingBracePattern.MatchString(currentLine)
		isCaseLabel := caseLabelPattern.MatchString(currentLine)
		isCommentOnlyLine := isCommentOnly(currentLine)
		isPackageDeclaration := isPackageLine(trimmedLine)
		currentInformation := lineInformationMap[lineNumber]
		currentStatementType := ""

		if currentInformation != nil {
			currentStatementType = currentInformation.statementType
		}

		if isPackageDeclaration {
			currentStatementType = "package"
		}

		needsBlankLine := false
		currentIsTopLevel := currentInformation != nil && currentInformation.isTopLevel
		currentIsScoped := currentInformation != nil && currentInformation.isScoped

		if len(resultLines) > 0 && !previousWasOpenBrace && !isClosingBrace && !isCaseLabel {
			if currentIsTopLevel && previousWasTopLevel && currentStatementType != previousStatementType {
				if f.CommentMode == CommentsFollow && previousWasComment {
				} else {
					needsBlankLine = true
				}
			} else if currentInformation != nil && (currentIsScoped || previousWasScoped) {
				if f.CommentMode == CommentsFollow && previousWasComment {
				} else {
					needsBlankLine = true
				}
			} else if currentStatementType != "" && previousStatementType != "" && currentStatementType != previousStatementType {
				if f.CommentMode == CommentsFollow && previousWasComment {
				} else {
					needsBlankLine = true
				}
			}

			if f.CommentMode == CommentsFollow && isCommentOnlyLine && !previousWasComment {
				nextLineNumber := f.findNextNonCommentLine(sourceLines, lineIndex+1)

				if nextLineNumber > 0 {
					nextInformation := lineInformationMap[nextLineNumber]

					if nextInformation != nil {
						nextIsTopLevel := nextInformation.isTopLevel
						nextIsScoped := nextInformation.isScoped

						if nextIsTopLevel && previousWasTopLevel && nextInformation.statementType != previousStatementType {
							needsBlankLine = true
						} else if nextIsScoped || previousWasScoped {
							needsBlankLine = true
						} else if nextInformation.statementType != "" && previousStatementType != "" && nextInformation.statementType != previousStatementType {
							needsBlankLine = true
						}
					}
				}
			}
		}

		if needsBlankLine {
			resultLines = append(resultLines, "")
		}

		resultLines = append(resultLines, currentLine)
		previousWasOpenBrace = isOpeningBrace || isCaseLabel
		previousWasComment = isCommentOnlyLine

		if currentInformation != nil {
			previousStatementType = currentInformation.statementType
			previousWasTopLevel = currentInformation.isTopLevel
			previousWasScoped = currentInformation.isScoped
		} else if currentStatementType != "" {
			previousStatementType = currentStatementType
			previousWasTopLevel = false
			previousWasScoped = false
		}
	}

	outputString := strings.Join(resultLines, "\n")

	if !strings.HasSuffix(outputString, "\n") {
		outputString += "\n"
	}

	return []byte(outputString)
}

func (f *Formatter) findNextNonCommentLine(sourceLines []string, startLineIndex int) int {
	for lineIndex := startLineIndex; lineIndex < len(sourceLines); lineIndex++ {
		trimmedLine := strings.TrimSpace(sourceLines[lineIndex])

		if trimmedLine == "" {
			continue
		}

		if isCommentOnly(sourceLines[lineIndex]) {
			continue
		}

		return lineIndex + 1
	}

	return 0
}
