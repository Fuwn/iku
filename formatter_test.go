package main

import (
	"testing"
)

func TestFormatRemovesExtraBlankLines(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatScopedStatements(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatNestedScopes(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatForLoop(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatSwitch(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatMultipleFunctions(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatTypeStruct(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatDifferentStatementTypes(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatConsecutiveIfs(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatCaseClauseStatements(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatDeferInlineFunc(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}

func TestFormatCaseClauseAssignments(t *testing.T) {
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
		t.Fatalf("Format error: %v", err)
	}

	if string(formattedResult) != expectedOutput {
		t.Errorf("got:\n%s\nwant:\n%s", formattedResult, expectedOutput)
	}
}
