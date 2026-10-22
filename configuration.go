package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Fuwn/iku/engine"
	"io/fs"
	"os"
	"strings"
)

type Configuration struct {
	GroupSingleLineFunctions bool   `json:"group_single_line_functions"`
	CommentMode              string `json:"comment_mode"`
}

func (configuration Configuration) formattingEngine() (engine.Engine, error) {
	formattingEngine := engine.Engine{SeparateSingleLineDefinitions: !configuration.GroupSingleLineFunctions}

	switch strings.ToLower(configuration.CommentMode) {
	case "", "follow":
		formattingEngine.CommentMode = engine.CommentsFollow
	case "precede":
		formattingEngine.CommentMode = engine.CommentsPrecede
	case "standalone":
		formattingEngine.CommentMode = engine.CommentsStandalone
	default:
		return formattingEngine, fmt.Errorf("invalid comment_mode: %q (use follow, precede, or standalone)", configuration.CommentMode)
	}

	return formattingEngine, nil
}

func loadFormattingEngine() (engine.Engine, error) {
	for _, fileName := range []string{".iku.json", "iku.json"} {
		fileData, readError := os.ReadFile(fileName)

		if errors.Is(readError, fs.ErrNotExist) {
			continue
		}

		if readError != nil {
			return engine.Engine{}, readError
		}

		formattingEngine, parseError := parseFormattingEngine(fileData)

		if parseError != nil {
			return engine.Engine{}, fmt.Errorf("%s: %w", fileName, parseError)
		}

		return formattingEngine, nil
	}

	return engine.Engine{}, nil
}

func parseFormattingEngine(fileData []byte) (engine.Engine, error) {
	configuration := Configuration{GroupSingleLineFunctions: true}
	decoder := json.NewDecoder(bytes.NewReader(fileData))

	decoder.DisallowUnknownFields()

	if decodingError := decoder.Decode(&configuration); decodingError != nil {
		return engine.Engine{}, decodingError
	}

	return configuration.formattingEngine()
}
