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

	formattingEngine := &engine.Engine{
		CommentMode:           f.CommentMode,
		GroupSingleLineScopes: f.Configuration.groupSingleLineFunctions(),
	}
	result := formattingEngine.FormatToBytes(events)

	if isEcmaScriptExtension(filepath.Ext(filename)) || bytes.Equal(normalizedSource, result) {
		return result, nil
	}

	// Regrouping declarations can change Go's column alignment and build-tag spacing.
	return format.Source(result)
}

func analyzeSource(source []byte, filename string) ([]byte, []engine.LineEvent, error) {
	if isEcmaScriptExtension(filepath.Ext(filename)) {
		return (&EcmaScriptAdapter{}).Analyze(source)
	}

	return (&GoAdapter{}).Analyze(source)
}

func isEcmaScriptExtension(extension string) bool {
	switch extension {
	case ".js", ".ts", ".jsx", ".tsx":
		return true
	default:
		return false
	}
}
