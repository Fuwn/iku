package main

import (
	"bytes"
	"fmt"
	"github.com/Fuwn/iku/engine"
	"go/format"
	"path/filepath"
)

type Formatter struct {
	Engine engine.Engine
}

func (formatter *Formatter) Format(source []byte, filename string) ([]byte, error) {
	extension := filepath.Ext(filename)

	switch {
	case isEcmaScriptExtension(extension):
		return formatter.Engine.Format(analyzeEcmaScript(source)), nil
	case extension == ".go" || extension == "":
		return formatter.formatGo(source)
	default:
		return nil, fmt.Errorf("unsupported file extension %q", extension)
	}
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
