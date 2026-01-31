package main

import (
	"strings"
	"testing"
)

func BenchmarkFormat_Small(b *testing.B) {
	input := []byte(`package main
func main() {
	x := 1
	y := 2
	if x > 0 {
		z := 3
	}
	a := 4
}
`)
	f := &Formatter{CommentMode: CommentsFollow}

	for b.Loop() {
		_, _ = f.Format(input)
	}
}

func BenchmarkFormat_Large(b *testing.B) {
	var sb strings.Builder

	sb.WriteString("package main\n\n")

	for i := range 100 {
		sb.WriteString("func foo")
		sb.WriteString(string(rune('A' + i%26)))
		sb.WriteString("() {\n")
		sb.WriteString("\tx := 1\n")
		sb.WriteString("\tif x > 0 {\n")
		sb.WriteString("\t\ty := 2\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\tz := 3\n")
		sb.WriteString("}\n\n")
	}

	input := []byte(sb.String())
	f := &Formatter{CommentMode: CommentsFollow}

	for b.Loop() {
		_, _ = f.Format(input)
	}
}
