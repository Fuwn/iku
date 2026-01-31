package main

import (
	"strings"
	"testing"
)

func BenchmarkFormat_Small(benchmarkRunner *testing.B) {
	inputSource := []byte(`package main
func main() {
	x := 1
	y := 2
	if x > 0 {
		z := 3
	}
	a := 4
}
`)
	formatter := &Formatter{CommentMode: CommentsFollow}

	for benchmarkRunner.Loop() {
		_, _ = formatter.Format(inputSource)
	}
}

func BenchmarkFormat_Large(benchmarkRunner *testing.B) {
	var sourceBuilder strings.Builder

	sourceBuilder.WriteString("package main\n\n")

	for functionIndex := range 100 {
		sourceBuilder.WriteString("func foo")
		sourceBuilder.WriteString(string(rune('A' + functionIndex%26)))
		sourceBuilder.WriteString("() {\n")
		sourceBuilder.WriteString("\tx := 1\n")
		sourceBuilder.WriteString("\tif x > 0 {\n")
		sourceBuilder.WriteString("\t\ty := 2\n")
		sourceBuilder.WriteString("\t}\n")
		sourceBuilder.WriteString("\tz := 3\n")
		sourceBuilder.WriteString("}\n\n")
	}

	inputSource := []byte(sourceBuilder.String())
	formatter := &Formatter{CommentMode: CommentsFollow}

	for benchmarkRunner.Loop() {
		_, _ = formatter.Format(inputSource)
	}
}
