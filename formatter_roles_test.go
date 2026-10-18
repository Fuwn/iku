package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func goFunctionBody(statements string) string {
	return "package example\n\nfunc example() {\n" + statements + "}\n"
}

func TestFormatStatementRoles(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		source   string
		expected string
	}{
		{
			name:     "Go bindings and reassignment",
			filename: "example.go",
			source:   goFunctionBody("\tvar total = 2\n\tcount := 1\n\tcount = total\n"),
			expected: goFunctionBody("\tvar total = 2\n\tcount := 1\n\n\tcount = total\n"),
		},
		{
			name:     "Go increments share the assignment role",
			filename: "example.go",
			source:   goFunctionBody("\tcount = 1\n\n\tcount++\n\tsave()\n"),
			expected: goFunctionBody("\tcount = 1\n\tcount++\n\n\tsave()\n"),
		},
		{
			name:     "Go local function bindings stay packed",
			filename: "example.go",
			source:   goFunctionBody("\tcallback := func() {\n\t\trun()\n\t}\n\n\tvar enabled = true\n"),
			expected: goFunctionBody("\tcallback := func() {\n\t\trun()\n\t}\n\tvar enabled = true\n"),
		},
		{
			name:     "Go top-level function bindings get separation",
			filename: "example.go",
			source:   "package example\n\nvar callback = func() {\n\trun()\n}\nvar enabled = true\n",
			expected: "package example\n\nvar callback = func() {\n\trun()\n}\n\nvar enabled = true\n",
		},
		{
			name:     "Go multiline initializers are not bodies",
			filename: "example.go",
			source:   "package example\n\nvar names = []string{\n\t\"first\",\n\t\"second\",\n}\n\nvar enabled = true\n",
			expected: "package example\n\nvar names = []string{\n\t\"first\",\n\t\"second\",\n}\nvar enabled = true\n",
		},
		{
			name:     "Go callback calls stay packed",
			filename: "example.go",
			source:   goFunctionBody("\tfirst(func() {\n\t\trun()\n\t})\n\n\tsecond(func() {\n\t\trun()\n\t})\n"),
			expected: goFunctionBody("\tfirst(func() {\n\t\trun()\n\t})\n\tsecond(func() {\n\t\trun()\n\t})\n"),
		},
		{
			name:     "Go assignment bodies get separation",
			filename: "example.go",
			source:   goFunctionBody("\tcallback = func() {\n\t\trun()\n\t}\n\tenabled = true\n"),
			expected: goFunctionBody("\tcallback = func() {\n\t\trun()\n\t}\n\n\tenabled = true\n"),
		},
		{
			name:     "Go comments attach after a blank line",
			filename: "example.go",
			source:   goFunctionBody("\tprepare()\n\t// The server requires preparation before writes.\n\tsave()\n"),
			expected: goFunctionBody("\tprepare()\n\n\t// The server requires preparation before writes.\n\tsave()\n"),
		},
		{
			name:     "Go one-line definitions align and cluster",
			filename: "example.go",
			source:   "package example\n\nfunc first() {}\n\nfunc second() {}\n",
			expected: "package example\n\nfunc first()  {}\nfunc second() {}\n",
		},
		{
			name:     "Go block-comment paragraphs survive",
			filename: "example.go",
			source:   "package example\n\n/*\nFirst paragraph.\n\nSecond paragraph.\n*/\nfunc run() {}\n",
			expected: "package example\n\n/*\nFirst paragraph.\n\nSecond paragraph.\n*/\nfunc run() {}\n",
		},
		{
			name:     "Go raw strings preserve paragraphs and delimiters",
			filename: "example.go",
			source:   goFunctionBody("\t// A lone backtick ` belongs to this comment.\n\tmessage := `first\n\nsecond { = }`\n\tvar enabled = true\n"),
			expected: goFunctionBody("\t// A lone backtick ` belongs to this comment.\n\tmessage := `first\n\nsecond { = }`\n\tvar enabled = true\n"),
		},
		{
			name:     "Go line directives do not change physical grouping",
			filename: "example.go",
			source:   "package example\n\n//line generated.go:100\nfunc example() {\n\tmessage := `first\n\nsecond`\n\tvar enabled = true\n\tenabled = false\n}\n",
			expected: "package example\n\n//line generated.go:100\nfunc example() {\n\tmessage := `first\n\nsecond`\n\tvar enabled = true\n\n\tenabled = false\n}\n",
		},
		{
			name:     "Go statements after block comments retain their role",
			filename: "example.go",
			source:   goFunctionBody("\t/* first\n\t   second */count := 1\n\tcount = 2\n"),
			expected: goFunctionBody("\t/* first\n\t   second */count := 1\n\n\tcount = 2\n"),
		},
		{
			name:     "Go build constraints keep their required separator",
			filename: "example.go",
			source:   "//go:build linux\n\npackage example\n\nvar enabled = true\n",
			expected: "//go:build linux\n\npackage example\n\nvar enabled = true\n",
		},
		{
			name:     "JavaScript declaration modifiers share one role",
			filename: "example.ts",
			source:   "export const first = 1;\n\nlet second = 2;\n\nvar third = 3;\n",
			expected: "export const first = 1;\nlet second = 2;\nvar third = 3;\n",
		},
		{
			name:     "JavaScript assignments differ from calls",
			filename: "example.ts",
			source:   "save();\ncount += 1;\nready = true;\nnotify();\n",
			expected: "save();\n\ncount += 1;\nready = true;\n\nnotify();\n",
		},
		{
			name:     "JavaScript operators in arguments are not assignments",
			filename: "example.ts",
			source:   "check(value === expected);\nlog(\"count = 1; { }\");\ncount++;\n",
			expected: "check(value === expected);\nlog(\"count = 1; { }\");\n\ncount++;\n",
		},
		{
			name:     "JavaScript top-level closure bindings get separation",
			filename: "example.ts",
			source:   "const callback = () => {\n  run();\n};\nconst enabled = true;\n",
			expected: "const callback = () => {\n  run();\n};\n\nconst enabled = true;\n",
		},
		{
			name:     "JavaScript local closure bindings stay packed",
			filename: "example.ts",
			source:   "function example() {\nconst callback = () => {\nrun();\n};\nlet enabled = true;\n}\n",
			expected: "function example() {\nconst callback = () => {\nrun();\n};\nlet enabled = true;\n}\n",
		},
		{
			name:     "JavaScript object initializers are not bodies",
			filename: "example.ts",
			source:   "const settings = {\n  enabled: true,\n  name: \"example\",\n};\n\nconst ready = true;\n",
			expected: "const settings = {\n  enabled: true,\n  name: \"example\",\n};\nconst ready = true;\n",
		},
		{
			name:     "JavaScript callback calls stay packed",
			filename: "example.ts",
			source:   "first(() => {\n  run();\n});\n\nsecond(() => {\n  run();\n});\ncount = 1;\n",
			expected: "first(() => {\n  run();\n});\nsecond(() => {\n  run();\n});\n\ncount = 1;\n",
		},
		{
			name:     "JavaScript comments attach after a blank line",
			filename: "example.ts",
			source:   "prepare();\n// The server requires preparation before writes.\nsave();\n",
			expected: "prepare();\n\n// The server requires preparation before writes.\nsave();\n",
		},
		{
			name:     "JavaScript template literals stay verbatim",
			filename: "example.ts",
			source:   "const message = `\nfirst\n\nsecond { = }\n`;\nlet enabled = true;\n",
			expected: "const message = `\nfirst\n\nsecond { = }\n`;\nlet enabled = true;\n",
		},
		{
			name:     "JavaScript regex literals do not close function bodies",
			filename: "example.ts",
			source:   "function example() {\n  const pattern = /[}=/]/;\n  let count = 1;\n  count >>= 1;\n  verify(pattern.test(\"}\"));\n}\n",
			expected: "function example() {\n  const pattern = /[}=/]/;\n  let count = 1;\n\n  count >>= 1;\n\n  verify(pattern.test(\"}\"));\n}\n",
		},
		{
			name:     "JavaScript Allman function bodies keep their owner",
			filename: "example.ts",
			source:   "function first() {}\nfunction second()\n{\n  run();\n}\nfunction third() {}\n",
			expected: "function first() {}\n\nfunction second()\n{\n  run();\n}\n\nfunction third() {}\n",
		},
		{
			name:     "JavaScript Allman control bodies keep their owner",
			filename: "example.ts",
			source:   "prepare();\nif (ready)\n{\n  run();\n}\nsave();\n",
			expected: "prepare();\n\nif (ready)\n{\n  run();\n}\n\nsave();\n",
		},
		{
			name:     "JavaScript Allman function expressions keep the binding",
			filename: "example.ts",
			source:   "const ready = true;\nconst callback = function()\n{\n  run();\n};\nlet enabled = true;\n",
			expected: "const ready = true;\n\nconst callback = function()\n{\n  run();\n};\n\nlet enabled = true;\n",
		},
		{
			name:     "JavaScript Allman callback calls stay packed",
			filename: "example.ts",
			source:   "first(function()\n{\n  run();\n});\n\nsecond();\n",
			expected: "first(function()\n{\n  run();\n});\nsecond();\n",
		},
		{
			name:     "JavaScript Allman methods have function bodies",
			filename: "example.ts",
			source:   "class Example {\n  first()\n  {\n    run();\n  }\n  second() {}\n}\n",
			expected: "class Example {\n  first()\n  {\n    run();\n  }\n\n  second() {}\n}\n",
		},
		{
			name:     "JavaScript methods have bodies and keep class context",
			filename: "example.ts",
			source:   "class Example {\n  first() {\n    run();\n  }\n  second() {\n    run();\n  }\n}\n",
			expected: "class Example {\n  first() {\n    run();\n  }\n\n  second() {\n    run();\n  }\n}\n",
		},
		{
			name:     "JavaScript assignment bodies get separation",
			filename: "example.ts",
			source:   "function example() {\n  callback = () => {\n    run();\n  };\n  enabled = true;\n}\n",
			expected: "function example() {\n  callback = () => {\n    run();\n  };\n\n  enabled = true;\n}\n",
		},
		{
			name:     "JavaScript nested callback calls stay packed",
			filename: "example.ts",
			source:   "function example() {\n  first(() => {\n    run();\n  });\n\n  second(() => {\n    run();\n  });\n}\n",
			expected: "function example() {\n  first(() => {\n    run();\n  });\n  second(() => {\n    run();\n  });\n}\n",
		},
		{
			name:     "JavaScript block-comment paragraphs survive",
			filename: "example.ts",
			source:   "/*\nFirst paragraph.\n\nSecond paragraph.\n*/\nrun();\n",
			expected: "/*\nFirst paragraph.\n\nSecond paragraph.\n*/\nrun();\n",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			formatter := &Formatter{CommentMode: CommentsFollow}
			result, formattingError := formatter.Format([]byte(testCase.source), testCase.filename)

			if formattingError != nil {
				t.Fatal(formattingError)
			}

			if string(result) != testCase.expected {
				t.Fatalf("got:\n%s\nwant:\n%s", result, testCase.expected)
			}

			secondResult, secondError := formatter.Format(result, testCase.filename)

			if secondError != nil {
				t.Fatal(secondError)
			}

			if !bytes.Equal(result, secondResult) {
				t.Fatalf("formatting is not idempotent:\nfirst:\n%s\nsecond:\n%s", result, secondResult)
			}
		})
	}
}

func TestFormatSingleLineFunctionConfiguration(t *testing.T) {
	cases := []struct {
		configuration string
		expected      string
	}{
		{`{}`, "package example\n\nfunc first()  {}\nfunc second() {}\n"},
		{`{"group_single_line_functions":true}`, "package example\n\nfunc first()  {}\nfunc second() {}\n"},
		{`{"group_single_line_functions":false}`, "package example\n\nfunc first() {}\n\nfunc second() {}\n"},
	}

	for _, testCase := range cases {
		var configuration Configuration

		if decodingError := json.Unmarshal([]byte(testCase.configuration), &configuration); decodingError != nil {
			t.Fatal(decodingError)
		}

		formatter := &Formatter{Configuration: configuration}
		result, formattingError := formatter.Format([]byte("package example\n\nfunc first() {}\nfunc second() {}\n"), "example.go")

		if formattingError != nil {
			t.Fatal(formattingError)
		}

		if string(result) != testCase.expected {
			t.Errorf("configuration %s: got %q; want %q", testCase.configuration, result, testCase.expected)
		}
	}
}
