package main

import (
	"encoding/json"
	"fmt"
	"github.com/Fuwn/iku/engine"
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

func loadConfiguration() Configuration {
	configuration := Configuration{GroupSingleLineFunctions: true}

	for _, fileName := range []string{".iku.json", "iku.json"} {
		fileData, readError := os.ReadFile(fileName)

		if readError != nil {
			continue
		}

		_ = json.Unmarshal(fileData, &configuration)

		break
	}

	return configuration
}
