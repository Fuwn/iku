package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Configuration struct {
	GroupSingleLineFunctions bool   `json:"group_single_line_functions"`
	CommentMode             string `json:"comment_mode"`
}

func (configuration Configuration) commentMode() (CommentMode, error) {
	switch strings.ToLower(configuration.CommentMode) {
	case "", "follow":
		return CommentsFollow, nil
	case "precede":
		return CommentsPrecede, nil
	case "standalone":
		return CommentsStandalone, nil
	default:
		return 0, fmt.Errorf("invalid comment_mode: %q (use follow, precede, or standalone)", configuration.CommentMode)
	}
}

func loadConfiguration() Configuration {
	var configuration Configuration

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
