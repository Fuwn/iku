package main

import (
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
	parsedFile, err := parser.ParseFile(tokenFileSet, "", formattedSource, parser.ParseComments|parser.SkipObjectResolution)

	if err != nil {
		return nil, nil, err
	}

	remainingSource := string(formattedSource)
	events := make([]engine.LineEvent, strings.Count(remainingSource, "\n")+1)

	indexGoStatements(tokenFileSet, parsedFile, events)

	for lineIndex := range events {
		currentLine, nextSource, _ := strings.Cut(remainingSource, "\n")
		event := &events[lineIndex]

		remainingSource = nextSource
		event.Content = currentLine
		event.TrimmedContent = strings.TrimSpace(currentLine)
		event.IsBlank = event.TrimmedContent == ""
		event.IsClosingBrace = isClosingBrace(currentLine)
		event.IsOpeningBrace = isOpeningBrace(currentLine)
		event.IsCaseLabel = isCaseLabel(currentLine)
		event.IsCommentOnly = isCommentOnly(currentLine)
		event.IsPackageDecl = isPackageLine(event.TrimmedContent)
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
				commentEnd := strings.Index(event.TrimmedContent, "*/")
				commentOnly := strings.HasPrefix(event.TrimmedContent, "/*") &&
					(commentEnd < 0 || strings.TrimSpace(event.TrimmedContent[commentEnd+2:]) == "")

				event.InRawString = commentLine != firstLine
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
			events[literalLine-1].InRawString = true
		}

		return false
	})

	return formattedSource, events, nil
}
