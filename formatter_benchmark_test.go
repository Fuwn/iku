package main

import (
	"strconv"
	"strings"
	"testing"
)

func BenchmarkFormatEcmaScript(b *testing.B) {
	for _, functionCount := range []int{1, 100} {
		b.Run(strconv.Itoa(functionCount), func(b *testing.B) {
			source := []byte(strings.Repeat(`function example() {
  const options = { ready: true };
  let count = 1;
  count += 1;
  first(() => {
    run(options);
  });
  second(/[}=/]/);
  return count;
}
`, functionCount))
			formatter := &Formatter{}

			b.SetBytes(int64(len(source)))

			for b.Loop() {
				_, _ = formatter.Format(source, "example.js")
			}
		})
	}
}

func BenchmarkFormatNested(b *testing.B) {
	source := []byte("package example\nfunc example() {\n" + strings.Repeat("if ready {\n", 20) + "run()\n" + strings.Repeat("}\n", 21))
	formatter := &Formatter{}

	for b.Loop() {
		_, _ = formatter.Format(source, "example.go")
	}
}

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
			formatter := &Formatter{}

			b.SetBytes(int64(len(source)))

			for b.Loop() {
				_, _ = formatter.Format(source, testCase.filename)
			}
		})
	}
}

func BenchmarkFormatArithmetic(b *testing.B) {
	source := []byte(strings.Repeat("function example(values) {\n  const first = values.total / values.count;\n  const second = first / 2;\n  return /item/.test(values.name);\n}\n", 100))
	formatter := &Formatter{}

	for b.Loop() {
		_, _ = formatter.Format(source, "example.js")
	}
}

func BenchmarkUnifiedDiff(b *testing.B) {
	for _, lineCount := range []int{1, 1000} {
		b.Run(strconv.Itoa(lineCount), func(b *testing.B) {
			original := []byte(strings.Repeat("first()\n", lineCount))
			formatted := []byte(strings.Repeat("first()\n\n", lineCount))

			for b.Loop() {
				_ = unifiedDiff("example.go", original, formatted)
			}
		})
	}
}
