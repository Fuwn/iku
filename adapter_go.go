package main

import (
	"bytes"
	"github.com/Fuwn/iku/engine"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

type GoAdapter struct{}

func (a *GoAdapter) Analyze(source []byte) ([]byte, []engine.LineEvent, error) {
	formattedSource, err := format.Source(source)

	if err != nil {
		return nil, nil, err
	}

	tokenFileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(tokenFileSet, "", formattedSource, parser.ParseComments)

	if err != nil {
		return nil, nil, err
	}

	formatter := &Formatter{}
	lineInformationMap := formatter.buildLineInfo(tokenFileSet, parsedFile)
	sourceByteLines := bytes.Split(formattedSource, []byte("\n"))
	events := make([]engine.LineEvent, len(sourceByteLines))
	blockCommentLines := make(map[int]bool)
	rawStringLines := make(map[int]bool)

	ast.Inspect(parsedFile, func(node ast.Node) bool {
		literal, isLiteral := node.(*ast.BasicLit)

		if !isLiteral || literal.Kind != token.STRING || !strings.HasPrefix(literal.Value, "`") {
			return true
		}

		firstLine := tokenFileSet.PositionFor(literal.Pos(), false).Line
		lastLine := tokenFileSet.PositionFor(literal.End(), false).Line

		for literalLine := firstLine + 1; literalLine <= lastLine; literalLine++ {
			rawStringLines[literalLine] = true
		}

		return false
	})

	for _, commentGroup := range parsedFile.Comments {
		for _, comment := range commentGroup.List {
			if !strings.HasPrefix(comment.Text, "/*") {
				continue
			}

			firstLine := tokenFileSet.PositionFor(comment.Pos(), false).Line
			lastLine := tokenFileSet.PositionFor(comment.End(), false).Line

			for commentLine := firstLine; commentLine <= lastLine; commentLine++ {
				blockCommentLines[commentLine] = commentLine != firstLine
			}
		}
	}

	for lineIndex, currentLineBytes := range sourceByteLines {
		currentLine := string(currentLineBytes)
		event := engine.NewLineEvent(currentLine)
		commentContinuation, isBlockComment := blockCommentLines[lineIndex+1]
		lineNumber := lineIndex + 1
		currentInformation := lineInformationMap[lineNumber]

		if currentInformation != nil {
			event.HasASTInfo = true
			event.StatementType = currentInformation.statementType
			event.IsTopLevel = currentInformation.isTopLevel
			event.IsScoped = currentInformation.isScoped
			event.IsSingleLineScope = currentInformation.isSingleLineScope
			event.IsStartLine = currentInformation.isStartLine
		}

		event.IsClosingBrace = isClosingBrace(currentLine)
		event.IsOpeningBrace = isOpeningBrace(currentLine)
		event.IsCaseLabel = isCaseLabel(currentLine)

		commentEnd := strings.Index(event.TrimmedContent, "*/")
		blockCommentOnly := isBlockComment && strings.HasPrefix(event.TrimmedContent, "/*") &&
			(commentEnd < 0 || strings.TrimSpace(event.TrimmedContent[commentEnd+2:]) == "")

		event.InRawString = rawStringLines[lineNumber] || commentContinuation
		event.IsCommentOnly = isCommentOnly(currentLine) || blockCommentOnly
		event.IsPackageDecl = isPackageLine(event.TrimmedContent)
		events[lineIndex] = event
	}

	return formattedSource, events, nil
}

func MapCommentMode(mode CommentMode) engine.CommentMode {
	switch mode {
	case CommentsFollow:
		return engine.CommentsFollow
	case CommentsPrecede:
		return engine.CommentsPrecede
	case CommentsStandalone:
		return engine.CommentsStandalone
	default:
		return engine.CommentsFollow
	}
}
