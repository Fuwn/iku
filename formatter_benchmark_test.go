package main

import (
	"strings"
	"testing"
)

func BenchmarkFormatVerbatim(b *testing.B) {
	paragraphs := strings.Repeat("literal { body }\n\n", 1000)
	cases := []struct {
		name     string
		filename string
		source   string
	}{
		{"GoRawString", "example.go", "package example\nvar message = `" + paragraphs + "`\nvar enabled = true\n"},
		{"GoBlockComment", "example.go", "package example\n/*\n" + paragraphs + "*/\nfunc example() {}\n"},
		{"JavaScriptTemplate", "example.js", "const message = `" + paragraphs + "`;\nlet enabled = true;\n"},
		{"JavaScriptBlockComment", "example.js", "/*\n" + paragraphs + "*/\nfunction example() {}\n"},
	}

	for _, testCase := range cases {
		b.Run(testCase.name, func(b *testing.B) {
			source := []byte(testCase.source)
			formatter := &Formatter{CommentMode: CommentsFollow}

			b.SetBytes(int64(len(source)))

			for b.Loop() {
				_, _ = formatter.Format(source, testCase.filename)
			}
		})
	}
}
