package main

import (
	"bytes"
	"github.com/Fuwn/iku/engine"
	"go/format"
	"path/filepath"
)

type Formatter struct {
	Engine engine.Engine
}

func (formatter *Formatter) Format(source []byte, filename string) ([]byte, error) {
	if isEcmaScriptExtension(filepath.Ext(filename)) {
		return formatter.Engine.Format(analyzeEcmaScript(source)), nil
	}

	return formatter.formatGo(source)
}

func (formatter *Formatter) formatGo(source []byte) ([]byte, error) {
	normalizedSource, events, err := analyzeGo(source)

	if err != nil {
		return nil, err
	}

	result := formatter.Engine.Format(events)

	if bytes.Equal(normalizedSource, result) {
		return result, nil
	}

	// Regrouping declarations can change Go's column alignment and build-tag spacing.
	return format.Source(result)
}

func isEcmaScriptExtension(extension string) bool {
	switch extension {
	case ".js", ".ts", ".jsx", ".tsx":
		return true
	default:
		return false
	}
}
