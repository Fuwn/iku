package main

import (
	"github.com/Fuwn/iku/engine"
	"testing"
)

type parityInput struct {
	name   string
	source string
}

var parityInputs = []parityInput{
	{
		name: "extra blank lines collapsed",
		source: `package main

func main() {
	x := 1


	y := 2
}
`,
	},
	{
		name: "scoped statements",
		source: `package main

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
		name: "nested scopes",
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
	},
	{
		name: "for loop",
		source: `package main

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
		name: "switch statement",
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
	},
	{
		name: "multiple functions",
		source: `package main

func foo() {
	x := 1
}


func bar() {
	y := 2
}
`,
	},
	{
		name: "type struct before var",
		source: `package main

type Foo struct {
	X int
}
var x = 1
`,
	},
	{
		name: "different statement types",
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
	},
	{
		name: "consecutive ifs",
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
	},
	{
		name: "case clause with scoped statement",
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
	},
	{
		name: "defer inline func",
		source: `package main

func main() {
	defer func() { _ = file.Close() }()
	fileInfo, err := file.Stat()
}
`,
	},
	{
		name: "case clause assignments only",
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
	},
	{
		name:   "raw string literal",
		source: "package main\n\nvar x = `\nline 1\n\nline 2\n`\nvar y = 1\n",
	},
	{
		name: "mixed top-level declarations",
		source: `package main

import "fmt"

const x = 1

var y = 2

type Z struct{}

func main() {
	fmt.Println(x, y)
}
`,
	},
	{
		name: "empty function body",
		source: `package main

func main() {
}
`,
	},
	{
		name: "comment before scoped statement",
		source: `package main

func main() {
	x := 1
	// this is a comment
	if x > 0 {
		y := 2
	}
}
`,
	},
	{
		name: "multiple blank lines between functions",
		source: `package main

func a() {}



func b() {}




func c() {}
`,
	},
	{
		name: "select statement",
		source: `package main

func main() {
	x := 1
	select {
	case <-ch:
		y := 2
	}
	z := 3
}
`,
	},
	{
		name: "range loop",
		source: `package main

func main() {
	items := []int{1, 2, 3}
	for _, item := range items {
		_ = item
	}
	done := true
}
`,
	},
	{
		name: "interface declaration",
		source: `package main

type Reader interface {
	Read(p []byte) (n int, err error)
}
var x = 1
`,
	},
}

func TestEngineParityWithFormatter(t *testing.T) {
	for _, commentMode := range []CommentMode{CommentsFollow, CommentsPrecede, CommentsStandalone} {
		for _, input := range parityInputs {
			name := input.name

			switch commentMode {
			case CommentsPrecede:
				name += "/precede"
			case CommentsStandalone:
				name += "/standalone"
			}

			t.Run(name, func(t *testing.T) {
				formatter := &Formatter{CommentMode: commentMode}
				oldResult, err := formatter.Format([]byte(input.source))

				if err != nil {
					t.Fatalf("old formatter error: %v", err)
				}

				adapter := &GoAdapter{}
				_, events, err := adapter.Analyze([]byte(input.source))

				if err != nil {
					t.Fatalf("adapter error: %v", err)
				}

				formattingEngine := &engine.Engine{CommentMode: MapCommentMode(commentMode)}
				newResult := formattingEngine.FormatToString(events)

				if string(oldResult) != newResult {
					t.Errorf("parity mismatch\nold:\n%s\nnew:\n%s", oldResult, newResult)
				}
			})
		}
	}
}
