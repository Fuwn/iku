package main

import (
	"fmt"
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

func BenchmarkFormatRust(b *testing.B) {
	for _, functionCount := range []int{1, 100} {
		b.Run(strconv.Itoa(functionCount), func(b *testing.B) {
			source := []byte(strings.Repeat(`/// Counts requests.
#[must_use]
pub fn example<'a>(router: &'a mut Router) -> Result<usize, Error> {
  let options = Options { ready: true };
  let mut count = 1;
  count += 1;
  router.mount("/", move |_| {
    Response::success(r#"{"ready": true}"#)
  });
  let size = match count {
    0 => return Err(Error::Empty),
    _ => count * 2,
  };
  Ok(size)
}
`, functionCount))
			formatter := &Formatter{}

			b.SetBytes(int64(len(source)))

			for b.Loop() {
				_, _ = formatter.Format(source, "example.rs")
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
		{"RustRawString", "example.rs", "const MESSAGE: &str = r#\"" + paragraphs + "\"#;\nlet enabled = true;\n"},
		{"RustBlockComment", "example.rs", "/*\n" + paragraphs + "*/\nfn example() {}\n"},
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

func BenchmarkFormatSmall(b *testing.B) {
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
	formatter := &Formatter{}

	for b.Loop() {
		_, _ = formatter.Format(inputSource, "test.go")
	}
}

func BenchmarkFormatLarge(b *testing.B) {
	var sourceBuilder strings.Builder

	sourceBuilder.WriteString("package main\n\n")

	for functionIndex := range 100 {
		fmt.Fprintf(&sourceBuilder, "func foo%d", functionIndex)
		sourceBuilder.WriteString("() {\n")
		sourceBuilder.WriteString("\tx := 1\n")
		sourceBuilder.WriteString("\tif x > 0 {\n")
		sourceBuilder.WriteString("\t\ty := 2\n")
		sourceBuilder.WriteString("\t}\n")
		sourceBuilder.WriteString("\tz := 3\n")
		sourceBuilder.WriteString("}\n\n")
	}

	inputSource := []byte(sourceBuilder.String())
	formatter := &Formatter{}

	for b.Loop() {
		_, _ = formatter.Format(inputSource, "test.go")
	}
}
