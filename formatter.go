package main

import (
	"go/format"
	"go/parser"
	"go/token"
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
