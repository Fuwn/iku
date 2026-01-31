package main

import (
	"testing"
)

func TestFormat_RemovesExtraBlankLines(testRunner *testing.T) {
	inputSource := `package main

func main() {
	x := 1


	y := 2
}
`
	expectedOutput := `package main

func main() {
	x := 1
	y := 2
}
`
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_AddsBlankLineAroundScopedStatements(testRunner *testing.T) {
	inputSource := `package main

func main() {
	x := 1
	if x > 0 {
		y := 2
	}
	z := 3
}
`
	expectedOutput := `package main

func main() {
	x := 1

	if x > 0 {
		y := 2
	}

	z := 3
}
`
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_NestedScopes(testRunner *testing.T) {
	inputSource := `package main

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
	expectedOutput := `package main

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
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_ForLoop(testRunner *testing.T) {
	inputSource := `package main

func main() {
	x := 1
	for i := 0; i < 10; i++ {
		y := i
	}
	z := 2
}
`
	expectedOutput := `package main

func main() {
	x := 1

	for i := 0; i < 10; i++ {
		y := i
	}

	z := 2
}
`
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_Switch(testRunner *testing.T) {
	inputSource := `package main

func main() {
	x := 1
	switch x {
	case 1:
		y := 2
	}
	z := 3
}
`
	expectedOutput := `package main

func main() {
	x := 1

	switch x {
	case 1:
		y := 2
	}

	z := 3
}
`
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_MultipleFunctions(testRunner *testing.T) {
	inputSource := `package main

func foo() {
	x := 1
}


func bar() {
	y := 2
}
`
	expectedOutput := `package main

func foo() {
	x := 1
}

func bar() {
	y := 2
}
`
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_TypeStruct(testRunner *testing.T) {
	inputSource := `package main

type Foo struct {
	X int
}
var x = 1
`
	expectedOutput := `package main

type Foo struct {
	X int
}

var x = 1
`
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_DifferentStatementTypes(testRunner *testing.T) {
	inputSource := `package main

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
	expectedOutput := `package main

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
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_ConsecutiveIfs(testRunner *testing.T) {
	inputSource := `package main

func main() {
	if err != nil {
		return
	}
	if x > 0 {
		y = 1
	}
}
`
	expectedOutput := `package main

func main() {
	if err != nil {
		return
	}

	if x > 0 {
		y = 1
	}
}
`
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_CaseClauseStatements(testRunner *testing.T) {
	inputSource := `package main

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
	expectedOutput := `package main

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
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_DeferWithInlineFunc(testRunner *testing.T) {
	inputSource := `package main

func main() {
	defer func() { _ = file.Close() }()
	fileInfo, err := file.Stat()
}
`
	expectedOutput := `package main

func main() {
	defer func() { _ = file.Close() }()

	fileInfo, err := file.Stat()
}
`
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormat_CaseClauseConsecutiveAssignments(testRunner *testing.T) {
	inputSource := `package main

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
	expectedOutput := `package main

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
	formatter := &Formatter{CommentMode: CommentsFollow}
	formattedResult, err := formatter.Format([]byte(inputSource))

	if err != nil {
		testRunner.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		testRunner.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}
