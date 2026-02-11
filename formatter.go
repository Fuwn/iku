package main

import (
	"github.com/Fuwn/iku/engine"
	"path/filepath"
)

type CommentMode int

const (
	CommentsFollow CommentMode = iota
	CommentsPrecede
	CommentsStandalone
)

type Formatter struct {
	CommentMode   CommentMode
	Configuration Configuration
}

type lineInformation struct {
	statementType string
	isTopLevel    bool
	isScoped      bool
	isStartLine   bool
}

func (f *Formatter) Format(source []byte, filename string) ([]byte, error) {
	_, events, err := analyzeSource(source, filename)

	if err != nil {
		return nil, err
	}

	formattingEngine := &engine.Engine{
		CommentMode:           MapCommentMode(f.CommentMode),
		GroupSingleLineScopes: f.Configuration.GroupSingleLineFunctions,
	}

	return formattingEngine.FormatToBytes(events), nil
}

func analyzeSource(source []byte, filename string) ([]byte, []engine.LineEvent, error) {
	switch filepath.Ext(filename) {
	case ".js", ".ts", ".jsx", ".tsx":
		return (&EcmaScriptAdapter{}).Analyze(source)
	default:
		return (&GoAdapter{}).Analyze(source)
	}
}
