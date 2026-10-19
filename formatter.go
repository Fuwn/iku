package main

import (
	"bytes"
	"github.com/Fuwn/iku/engine"
	"go/format"
	"path/filepath"
)

type CommentMode = engine.CommentMode

const (
	CommentsFollow     = engine.CommentsFollow
	CommentsPrecede    = engine.CommentsPrecede
	CommentsStandalone = engine.CommentsStandalone
)

type Formatter struct {
	CommentMode   CommentMode
	Configuration Configuration
}

func (f *Formatter) Format(source []byte, filename string) ([]byte, error) {
	normalizedSource, events, err := analyzeSource(source, filename)

	if err != nil {
		return nil, err
	}

	commentMode := f.CommentMode

	if commentMode < CommentsFollow || commentMode > CommentsStandalone {
		commentMode = CommentsFollow
	}

	formattingEngine := &engine.Engine{
		CommentMode:           commentMode,
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

		lineBreakStart := outputPosition

		for outputPosition < len(result) && result[outputPosition] == '\n' {
			outputPosition++
		}

		if hadBlankLine && outputPosition-lineBreakStart < 2 {
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
