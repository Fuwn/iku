package main

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strings"
)

var (
	closingBracePattern = regexp.MustCompile(`^\s*[\}\)]`)
	openingBracePattern = regexp.MustCompile(`[\{\(]\s*$`)
	caseLabelPattern    = regexp.MustCompile(`^\s*(case\s|default\s*:)|(^\s+.*:\s*$)`)
)

func isCommentOnly(line string) bool {
	for index := range len(line) {
		character := line[index]

		if character == ' ' || character == '\t' {
			continue
		}

		return len(line) > index+1 && line[index] == '/' && line[index+1] == '/'
	}

	return false
}

func isPackageLine(trimmed string) bool {
	return len(trimmed) > 8 && trimmed[:8] == "package "
}

func countRawStringDelimiters(line string) int {
	count := 0
	inString := false
	inCharacter := false

	for index := 0; index < len(line); index++ {
		character := line[index]

		if inCharacter {
			if character == '\\' && index+1 < len(line) {
				index++

				continue
			}

			if character == '\'' {
				inCharacter = false
			}

			continue
		}

		if inString {
			if character == '\\' && index+1 < len(line) {
				index++

				continue
			}

			if character == '"' {
				inString = false
			}

			continue
		}

		if character == '\'' {
			inCharacter = true

			continue
		}

		if character == '"' {
			inString = true

			continue
		}

		if character == '`' {
			count++
		}
	}

	return count
}

type CommentMode int

const (
	CommentsFollow CommentMode = iota
	CommentsPrecede
	CommentsStandalone
)

type Formatter struct {
	CommentMode CommentMode
}

type lineInfo struct {
	statementType string
	isTopLevel    bool
	isScoped      bool
	isStartLine   bool
}

func (f *Formatter) Format(source []byte) ([]byte, error) {
	formatted, err := format.Source(source)

	if err != nil {
		return nil, err
	}

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "", formatted, parser.ParseComments)

	if err != nil {
		return nil, err
	}

	lineInfoMap := f.buildLineInfo(fileSet, file)

	return f.rewrite(formatted, lineInfoMap), nil
}

func isGenDeclScoped(genDecl *ast.GenDecl) bool {
	for _, spec := range genDecl.Specs {
		if typeSpec, ok := spec.(*ast.TypeSpec); ok {
			switch typeSpec.Type.(type) {
			case *ast.StructType, *ast.InterfaceType:
				return true
			}
		}
	}

	return false
}

func (f *Formatter) buildLineInfo(fileSet *token.FileSet, file *ast.File) map[int]*lineInfo {
	lineInfoMap := make(map[int]*lineInfo)
	tokenFile := fileSet.File(file.Pos())

	if tokenFile == nil {
		return lineInfoMap
	}

	for _, declaration := range file.Decls {
		startLine := tokenFile.Line(declaration.Pos())
		endLine := tokenFile.Line(declaration.End())
		typeName := ""
		isScoped := false

		switch declarationType := declaration.(type) {
		case *ast.GenDecl:
			typeName = declarationType.Tok.String()
			isScoped = isGenDeclScoped(declarationType)
		case *ast.FuncDecl:
			typeName = "func"
			isScoped = true
		default:
			typeName = reflect.TypeOf(declaration).String()
		}

		lineInfoMap[startLine] = &lineInfo{statementType: typeName, isTopLevel: true, isScoped: isScoped, isStartLine: true}
		lineInfoMap[endLine] = &lineInfo{statementType: typeName, isTopLevel: true, isScoped: isScoped, isStartLine: false}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		switch typedNode := node.(type) {
		case *ast.BlockStmt:
			f.processBlock(tokenFile, typedNode, lineInfoMap)
		case *ast.CaseClause:
			f.processStatementList(tokenFile, typedNode.Body, lineInfoMap)
		case *ast.CommClause:
			f.processStatementList(tokenFile, typedNode.Body, lineInfoMap)
		}

		return true
	})

	return lineInfoMap
}

func (f *Formatter) processBlock(tokenFile *token.File, block *ast.BlockStmt, lineInfoMap map[int]*lineInfo) {
	if block == nil {
		return
	}

	f.processStatementList(tokenFile, block.List, lineInfoMap)
}

func (f *Formatter) processStatementList(tokenFile *token.File, statements []ast.Stmt, lineInfoMap map[int]*lineInfo) {
	for _, statement := range statements {
		startLine := tokenFile.Line(statement.Pos())
		endLine := tokenFile.Line(statement.End())
		typeName := ""
		isScoped := false

		switch statementType := statement.(type) {
		case *ast.DeclStmt:
			if genericDeclaration, ok := statementType.Decl.(*ast.GenDecl); ok {
				typeName = genericDeclaration.Tok.String()
			} else {
				typeName = reflect.TypeOf(statement).String()
			}
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
			*ast.TypeSwitchStmt, *ast.SelectStmt, *ast.BlockStmt:
			typeName = reflect.TypeOf(statement).String()
			isScoped = true
		default:
			typeName = reflect.TypeOf(statement).String()
		}

		existingStart := lineInfoMap[startLine]

		if existingStart == nil || !existingStart.isStartLine {
			lineInfoMap[startLine] = &lineInfo{statementType: typeName, isTopLevel: false, isScoped: isScoped, isStartLine: true}
		}

		existingEnd := lineInfoMap[endLine]

		if existingEnd == nil || !existingEnd.isStartLine {
			lineInfoMap[endLine] = &lineInfo{statementType: typeName, isTopLevel: false, isScoped: isScoped, isStartLine: false}
		}

		switch typedStatement := statement.(type) {
		case *ast.IfStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInfoMap)

			if typedStatement.Else != nil {
				if elseBlock, isBlockStatement := typedStatement.Else.(*ast.BlockStmt); isBlockStatement {
					f.processBlock(tokenFile, elseBlock, lineInfoMap)
				} else if elseIfStatement, isIfStatement := typedStatement.Else.(*ast.IfStmt); isIfStatement {
					f.processIfStatement(tokenFile, elseIfStatement, lineInfoMap)
				}
			}
		case *ast.ForStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInfoMap)
		case *ast.RangeStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInfoMap)
		case *ast.SwitchStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInfoMap)
		case *ast.TypeSwitchStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInfoMap)
		case *ast.SelectStmt:
			f.processBlock(tokenFile, typedStatement.Body, lineInfoMap)
		case *ast.BlockStmt:
			f.processBlock(tokenFile, typedStatement, lineInfoMap)
		}
	}
}

func (f *Formatter) processIfStatement(tokenFile *token.File, ifStatement *ast.IfStmt, lineInfoMap map[int]*lineInfo) {
	startLine := tokenFile.Line(ifStatement.Pos())
	endLine := tokenFile.Line(ifStatement.End())
	existingStart := lineInfoMap[startLine]

	if existingStart == nil || !existingStart.isStartLine {
		lineInfoMap[startLine] = &lineInfo{statementType: "*ast.IfStmt", isTopLevel: false, isScoped: true, isStartLine: true}
	}

	existingEnd := lineInfoMap[endLine]

	if existingEnd == nil || !existingEnd.isStartLine {
		lineInfoMap[endLine] = &lineInfo{statementType: "*ast.IfStmt", isTopLevel: false, isScoped: true, isStartLine: false}
	}

	f.processBlock(tokenFile, ifStatement.Body, lineInfoMap)

	if ifStatement.Else != nil {
		if elseBlock, isBlockStatement := ifStatement.Else.(*ast.BlockStmt); isBlockStatement {
			f.processBlock(tokenFile, elseBlock, lineInfoMap)
		} else if elseIfStatement, isIfStatement := ifStatement.Else.(*ast.IfStmt); isIfStatement {
			f.processIfStatement(tokenFile, elseIfStatement, lineInfoMap)
		}
	}
}

func (f *Formatter) rewrite(source []byte, lineInfoMap map[int]*lineInfo) []byte {
	lines := strings.Split(string(source), "\n")
	result := make([]string, 0, len(lines))
	previousWasOpenBrace := false
	previousType := ""
	previousWasComment := false
	previousWasTopLevel := false
	previousWasScoped := false
	insideRawString := false

	for index, line := range lines {
		backtickCount := countRawStringDelimiters(line)
		wasInsideRawString := insideRawString

		if backtickCount%2 == 1 {
			insideRawString = !insideRawString
		}

		if wasInsideRawString {
			result = append(result, line)

			continue
		}

		lineNumber := index + 1
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		isClosingBrace := closingBracePattern.MatchString(line)
		isOpeningBrace := openingBracePattern.MatchString(line)
		isCaseLabel := caseLabelPattern.MatchString(line)
		isCommentOnlyLine := isCommentOnly(line)
		isPackageLine := isPackageLine(trimmed)
		info := lineInfoMap[lineNumber]
		currentType := ""

		if info != nil {
			currentType = info.statementType
		}

		if isPackageLine {
			currentType = "package"
		}

		needsBlank := false
		currentIsTopLevel := info != nil && info.isTopLevel
		currentIsScoped := info != nil && info.isScoped

		if len(result) > 0 && !previousWasOpenBrace && !isClosingBrace && !isCaseLabel {
			if currentIsTopLevel && previousWasTopLevel && currentType != previousType {
				if f.CommentMode == CommentsFollow && previousWasComment {
				} else {
					needsBlank = true
				}
			} else if info != nil && (currentIsScoped || previousWasScoped) {
				if f.CommentMode == CommentsFollow && previousWasComment {
				} else {
					needsBlank = true
				}
			} else if currentType != "" && previousType != "" && currentType != previousType {
				if f.CommentMode == CommentsFollow && previousWasComment {
				} else {
					needsBlank = true
				}
			}

			if f.CommentMode == CommentsFollow && isCommentOnlyLine && !previousWasComment {
				nextLineNumber := f.findNextNonCommentLine(lines, index+1)

				if nextLineNumber > 0 {
					nextInfo := lineInfoMap[nextLineNumber]

					if nextInfo != nil {
						nextIsTopLevel := nextInfo.isTopLevel
						nextIsScoped := nextInfo.isScoped

						if nextIsTopLevel && previousWasTopLevel && nextInfo.statementType != previousType {
							needsBlank = true
						} else if nextIsScoped || previousWasScoped {
							needsBlank = true
						} else if nextInfo.statementType != "" && previousType != "" && nextInfo.statementType != previousType {
							needsBlank = true
						}
					}
				}
			}
		}

		if needsBlank {
			result = append(result, "")
		}

		result = append(result, line)
		previousWasOpenBrace = isOpeningBrace || isCaseLabel
		previousWasComment = isCommentOnlyLine

		if info != nil {
			previousType = info.statementType
			previousWasTopLevel = info.isTopLevel
			previousWasScoped = info.isScoped
		} else if currentType != "" {
			previousType = currentType
			previousWasTopLevel = false
			previousWasScoped = false
		}
	}

	output := strings.Join(result, "\n")

	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return []byte(output)
}

func (f *Formatter) findNextNonCommentLine(lines []string, startIndex int) int {
	for index := startIndex; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])

		if trimmed == "" {
			continue
		}

		if isCommentOnly(lines[index]) {
			continue
		}

		return index + 1
	}

	return 0
}
