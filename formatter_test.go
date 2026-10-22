package main

import (
	"bytes"
	"go/format"
	"strings"
	"testing"
)

func goFunctionBody(statements string) string {
	return "package example\n\nfunc example() {\n" + statements + "}\n"
}

func TestFormatDeepNesting(t *testing.T) {
	const nestingDepth = 600
	nestedStatements := strings.Repeat("if true {\n", nestingDepth) + "return\n" + strings.Repeat("}\n", nestingDepth)
	source := []byte(goFunctionBody(nestedStatements))
	expected, err := format.Source(source)

	if err != nil {
		t.Fatal(err)
	}

	formatter := &Formatter{}
	actual, err := formatter.Format(source, "example.go")

	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(actual, expected) {
		t.Fatal("Deeply nested block formatting differs from gofmt.")
	}
}

type formattingCase struct {
	name     string
	filename string
	source   string
	expected string
}

func assertFormatting(t *testing.T, formatter *Formatter, testCase formattingCase) {
	t.Helper()

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
}

func TestFormatStatementRoles(t *testing.T) {
	cases := []formattingCase{
		{
			name:     "Go removes extra blank lines",
			filename: "example.go",
			source: `package main

func main() {
	x := 1


	y := 2
}
`,
			expected: `package main

func main() {
	x := 1
	y := 2
}
`,
		},
		{
			name:     "Go scoped statements",
			filename: "example.go",
			source: `package main

func main() {
	x := 1
	if x > 0 {
		y := 2
	}
	z := 3
}
`,
			expected: `package main

func main() {
	x := 1

	if x > 0 {
		y := 2
	}

	z := 3
}
`,
		},
		{
			name:     "Go nested scopes",
			filename: "example.go",
			source: `package main

func main() {
	if true {
		x := 1
		if false {
			y := 2
		}
		z := 3
	}
}
`,
			expected: `package main

func main() {
	if true {
		x := 1

		if false {
			y := 2
		}

		z := 3
	}
}
`,
		},
		{
			name:     "Go for loop",
			filename: "example.go",
			source: `package main

func main() {
	x := 1
	for i := 0; i < 10; i++ {
		y := i
	}
	z := 2
}
`,
			expected: `package main

func main() {
	x := 1

	for i := 0; i < 10; i++ {
		y := i
	}

	z := 2
}
`,
		},
		{
			name:     "Go switch",
			filename: "example.go",
			source: `package main

func main() {
	x := 1
	switch x {
	case 1:
		y := 2
	}
	z := 3
}
`,
			expected: `package main

func main() {
	x := 1

	switch x {
	case 1:
		y := 2
	}

	z := 3
}
`,
		},
		{
			name:     "Go multiple functions",
			filename: "example.go",
			source: `package main

func foo() {
	x := 1
}


func bar() {
	y := 2
}
`,
			expected: `package main

func foo() {
	x := 1
}

func bar() {
	y := 2
}
`,
		},
		{
			name:     "Go type struct",
			filename: "example.go",
			source: `package main

type Foo struct {
	X int
}
var x = 1
`,
			expected: `package main

type Foo struct {
	X int
}

var x = 1
`,
		},
		{
			name:     "Go different statement types",
			filename: "example.go",
			source: `package main

func main() {
	x := 1
	y := 2
	var a = 3
	defer cleanup()
	defer cleanup2()
	go worker()
	return
}
`,
			expected: `package main

func main() {
	x := 1
	y := 2
	var a = 3

	defer cleanup()
	defer cleanup2()

	go worker()

	return
}
`,
		},
		{
			name:     "Go consecutive ifs",
			filename: "example.go",
			source: `package main

func main() {
	if err != nil {
		return
	}
	if x > 0 {
		y = 1
	}
}
`,
			expected: `package main

func main() {
	if err != nil {
		return
	}

	if x > 0 {
		y = 1
	}
}
`,
		},
		{
			name:     "Go case clause statements",
			filename: "example.go",
			source: `package main

func main() {
	switch x {
	case 1:
		foo()
		if err != nil {
			return
		}
	}
}
`,
			expected: `package main

func main() {
	switch x {
	case 1:
		foo()

		if err != nil {
			return
		}
	}
}
`,
		},
		{
			name:     "Go defer inline func",
			filename: "example.go",
			source: `package main

func main() {
	defer func() { _ = file.Close() }()
	fileInfo, err := file.Stat()
}
`,
			expected: `package main

func main() {
	defer func() { _ = file.Close() }()

	fileInfo, err := file.Stat()
}
`,
		},
		{
			name:     "Go case clause assignments",
			filename: "example.go",
			source: `package main

func main() {
	switch x {
	case "user":
		roleStyle = UserStyle
		contentStyle = ContentStyle
		prefix = "You"
	case "assistant":
		roleStyle = AssistantStyle
	}
}
`,
			expected: `package main

func main() {
	switch x {
	case "user":
		roleStyle = UserStyle
		contentStyle = ContentStyle
		prefix = "You"
	case "assistant":
		roleStyle = AssistantStyle
	}
}
`,
		},
		{
			name:     "Go nested labels are separated like other statements",
			filename: "example.go",
			source:   goFunctionBody("\tfor {\n\t\tvalue := 1\n\tinner:\n\t\tfor {\n\t\t\tbreak inner\n\t\t}\n\t}\n"),
			expected: goFunctionBody("\tfor {\n\t\tvalue := 1\n\n\tinner:\n\t\tfor {\n\t\t\tbreak inner\n\t\t}\n\t}\n"),
		},
		{
			name:     "Go comments ending in colons attach to the next statement",
			filename: "example.go",
			source:   goFunctionBody("\trun()\n\t// Output:\n\t// done\n\tstop()\n"),
			expected: goFunctionBody("\trun()\n\n\t// Output:\n\t// done\n\tstop()\n"),
		},
		{
			name:     "Go multiline case headers open their clause body",
			filename: "example.go",
			source:   goFunctionBody("\tswitch {\n\tcase first &&\n\t\tsecond:\n\t\t// Explain the branch.\n\t\trun()\n\t}\n"),
			expected: goFunctionBody("\tswitch {\n\tcase first &&\n\t\tsecond:\n\t\t// Explain the branch.\n\t\trun()\n\t}\n"),
		},
		{
			name:     "Go raw strings and block comments can share a line",
			filename: "example.go",
			source:   "package example\nfunc example() {\nmessage := `first\nsecond` /* first\nsecond */\nvar enabled = true\nenabled = false\n}\n",
			expected: "package example\n\nfunc example() {\n\tmessage := `first\nsecond` /* first\n\tsecond */\n\tvar enabled = true\n\n\tenabled = false\n}\n",
		},
		{
			name:     "Go adjacent block comments retain statement roles",
			filename: "example.go",
			source:   "package example\nfunc example() {\n/* first\n second */ /* third */ count := 1\ncount = 2\n}\n",
			expected: "package example\n\nfunc example() {\n\t/* first\n\t   second */ /* third */\n\n\tcount := 1\n\n\tcount = 2\n}\n",
		},
		{
			name:     "Go else-if branches keep statement roles",
			filename: "example.go",
			source:   goFunctionBody("\tif first {\n\t\trun()\n\t} else if second {\n\t\tcount := 1\n\t\tvar enabled = true\n\t\tcount = 2\n\t\tsave()\n\t} else {\n\t\tstop()\n\t}\n\tnotify()\n"),
			expected: goFunctionBody("\tif first {\n\t\trun()\n\t} else if second {\n\t\tcount := 1\n\t\tvar enabled = true\n\n\t\tcount = 2\n\n\t\tsave()\n\t} else {\n\t\tstop()\n\t}\n\n\tnotify()\n"),
		},
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
			assertFormatting(t, &Formatter{}, testCase)
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
		formattingEngine, configurationError := parseFormattingEngine([]byte(testCase.configuration))

		if configurationError != nil {
			t.Fatal(configurationError)
		}

		formatter := &Formatter{Engine: formattingEngine}
		result, formattingError := formatter.Format([]byte("package example\n\nfunc first() {}\nfunc second() {}\n"), "example.go")

		if formattingError != nil {
			t.Fatal(formattingError)
		}

		if string(result) != testCase.expected {
			t.Errorf("configuration %s: got %q; want %q", testCase.configuration, result, testCase.expected)
		}
	}
}

func TestFormatRejectsUnsupportedExtension(t *testing.T) {
	if _, formattingError := (&Formatter{}).Format([]byte("print(1)\n"), "example.py"); formattingError == nil {
		t.Fatal("Expected an unsupported extension error.")
	}
}
