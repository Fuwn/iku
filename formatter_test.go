package main

import (
	"testing"
)

func TestFormat_RemovesExtraBlankLines(t *testing.T) {
	input := `package main

func main() {
	x := 1


	y := 2
}
`
	expected := `package main

func main() {
	x := 1
	y := 2
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_AddsBlankLineAroundScopedStatements(t *testing.T) {
	input := `package main

func main() {
	x := 1
	if x > 0 {
		y := 2
	}
	z := 3
}
`
	expected := `package main

func main() {
	x := 1

	if x > 0 {
		y := 2
	}

	z := 3
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_NestedScopes(t *testing.T) {
	input := `package main

func main() {
	if true {
		x := 1
		if false {
			y := 2
		}
		z := 3
	}
}
`
	expected := `package main

func main() {
	if true {
		x := 1

		if false {
			y := 2
		}

		z := 3
	}
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_ForLoop(t *testing.T) {
	input := `package main

func main() {
	x := 1
	for i := 0; i < 10; i++ {
		y := i
	}
	z := 2
}
`
	expected := `package main

func main() {
	x := 1

	for i := 0; i < 10; i++ {
		y := i
	}

	z := 2
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_Switch(t *testing.T) {
	input := `package main

func main() {
	x := 1
	switch x {
	case 1:
		y := 2
	}
	z := 3
}
`
	expected := `package main

func main() {
	x := 1

	switch x {
	case 1:
		y := 2
	}

	z := 3
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_MultipleFunctions(t *testing.T) {
	input := `package main

func foo() {
	x := 1
}


func bar() {
	y := 2
}
`
	expected := `package main

func foo() {
	x := 1
}

func bar() {
	y := 2
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_TypeStruct(t *testing.T) {
	input := `package main

type Foo struct {
	X int
}
var x = 1
`
	expected := `package main

type Foo struct {
	X int
}

var x = 1
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_DifferentStatementTypes(t *testing.T) {
	input := `package main

func main() {
	x := 1
	y := 2
	var a = 3
	defer cleanup()
	defer cleanup2()
	go worker()
	return
}
`
	expected := `package main

func main() {
	x := 1
	y := 2

	var a = 3

	defer cleanup()
	defer cleanup2()

	go worker()

	return
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_ConsecutiveIfs(t *testing.T) {
	input := `package main

func main() {
	if err != nil {
		return
	}
	if x > 0 {
		y = 1
	}
}
`
	expected := `package main

func main() {
	if err != nil {
		return
	}

	if x > 0 {
		y = 1
	}
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_CaseClauseStatements(t *testing.T) {
	input := `package main

func main() {
	switch x {
	case 1:
		foo()
		if err != nil {
			return
		}
	}
}
`
	expected := `package main

func main() {
	switch x {
	case 1:
		foo()

		if err != nil {
			return
		}
	}
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_DeferWithInlineFunc(t *testing.T) {
	input := `package main

func main() {
	defer func() { _ = file.Close() }()
	fileInfo, err := file.Stat()
}
`
	expected := `package main

func main() {
	defer func() { _ = file.Close() }()

	fileInfo, err := file.Stat()
}
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestFormat_CaseClauseConsecutiveAssignments(t *testing.T) {
	input := `package main

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
`
	expected := `package main

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
`
	f := &Formatter{CommentMode: CommentsFollow}
	result, err := f.Format([]byte(input))

	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	if string(result) != expected {
		t.Errorf("got:\n%s\nwant:\n%s", result, expected)
	}
}
