package main

import (
	"bytes"
	"github.com/Fuwn/iku/engine"
	"go/format"
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
	statementType     string
	isTopLevel        bool
	isScoped          bool
	isSingleLineScope bool
	isStartLine       bool
}

func (f *Formatter) Format(source []byte, filename string) ([]byte, error) {
	normalizedSource, events, err := analyzeSource(source, filename)

	if err != nil {
		return nil, err
	}

	formattingEngine := &engine.Engine{
		CommentMode:           MapCommentMode(f.CommentMode),
		GroupSingleLineScopes: f.Configuration.groupSingleLineFunctions(),
	}
	result := formattingEngine.FormatToBytes(events)

	switch filepath.Ext(filename) {
	case ".js", ".ts", ".jsx", ".tsx":
		return result, nil
	}

	if bytes.Equal(normalizedSource, result) || !needsGoNormalization(events, result) {
		return result, nil
	}

	// Regrouping declarations can change Go's column alignment and build-tag spacing.
	return format.Source(result)
}

func needsGoNormalization(events []engine.LineEvent, result []byte) bool {
	outputPosition := 0
	hadBlankLine := false

	for _, event := range events {
		if event.IsCommentOnly || (event.StatementType == "function" && event.IsSingleLineScope) {
			return true
		}

		if event.IsBlank && !event.InRawString {
			hadBlankLine = true

			continue
		}

		lineBreaks := 0

		for outputPosition < len(result) && result[outputPosition] == '\n' {
			outputPosition++
			lineBreaks++
		}

		if hadBlankLine && lineBreaks < 2 {
			return true
		}

		hadBlankLine = false
		outputPosition += len(event.Content)
	}

	return false
}

func analyzeSource(source []byte, filename string) ([]byte, []engine.LineEvent, error) {
	switch filepath.Ext(filename) {
	case ".js", ".ts", ".jsx", ".tsx":
		return (&EcmaScriptAdapter{}).Analyze(source)
	default:
		return (&GoAdapter{}).Analyze(source)
	}
}
