package main

import (
	"bytes"
	"github.com/Fuwn/iku/engine"
	"go/format"
	"go/parser"
	"go/token"
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
	insideRawString := false

	for lineIndex, currentLineBytes := range sourceByteLines {
		currentLine := string(currentLineBytes)
		backtickCount := countRawStringDelimiters(currentLine)
		wasInsideRawString := insideRawString

		if backtickCount%2 == 1 {
			insideRawString = !insideRawString
		}

		event := engine.NewLineEvent(currentLine)

		if wasInsideRawString {
			event.InRawString = true
			events[lineIndex] = event

			continue
		}

		if event.IsBlank {
			events[lineIndex] = event

			continue
		}

		lineNumber := lineIndex + 1
		currentInformation := lineInformationMap[lineNumber]

		if currentInformation != nil {
			event.HasASTInfo = true
			event.StatementType = currentInformation.statementType
			event.IsTopLevel = currentInformation.isTopLevel
			event.IsScoped = currentInformation.isScoped
			event.IsStartLine = currentInformation.isStartLine
		}

		event.IsClosingBrace = isClosingBrace(currentLine)
		event.IsOpeningBrace = isOpeningBrace(currentLine)
		event.IsCaseLabel = isCaseLabel(currentLine)
		event.IsCommentOnly = isCommentOnly(currentLine)
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
