package engine

import (
	"strings"
	"testing"
)

func formatResult(formattingEngine *Engine, events []LineEvent) string {
	result := formattingEngine.FormatToString(events)

	return strings.TrimSuffix(result, "\n")
}

func TestEngineCollapsesBlanks(t *testing.T) {
	events := []LineEvent{
		{Content: "\tx := 1", TrimmedContent: "x := 1", HasASTInfo: true, StatementType: "*ast.AssignStmt", IsStartLine: true},
		{Content: "", TrimmedContent: "", IsBlank: true},
		{Content: "", TrimmedContent: "", IsBlank: true},
		{Content: "\ty := 2", TrimmedContent: "y := 2", HasASTInfo: true, StatementType: "*ast.AssignStmt", IsStartLine: true},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)

	if result != "\tx := 1\n\ty := 2" {
		t.Errorf("expected blanks collapsed, got:\n%s", result)
	}
}

func TestEngineScopeBoundary(t *testing.T) {
	events := []LineEvent{
		{Content: "\tx := 1", TrimmedContent: "x := 1", HasASTInfo: true, StatementType: "*ast.AssignStmt"},
		{Content: "\tif x > 0 {", TrimmedContent: "if x > 0 {", HasASTInfo: true, StatementType: "*ast.IfStmt", IsScoped: true, IsStartLine: true, IsOpeningBrace: true},
		{Content: "\t\ty := 2", TrimmedContent: "y := 2", HasASTInfo: true, StatementType: "*ast.AssignStmt"},
		{Content: "\t}", TrimmedContent: "}", IsClosingBrace: true, HasASTInfo: true, StatementType: "*ast.IfStmt", IsScoped: true},
		{Content: "\tz := 3", TrimmedContent: "z := 3", HasASTInfo: true, StatementType: "*ast.AssignStmt"},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)
	expected := "\tx := 1\n\n\tif x > 0 {\n\t\ty := 2\n\t}\n\n\tz := 3"

	if result != expected {
		t.Errorf("expected scope boundaries, got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestEngineStatementTypeTransition(t *testing.T) {
	events := []LineEvent{
		{Content: "\tx := 1", TrimmedContent: "x := 1", HasASTInfo: true, StatementType: "*ast.AssignStmt"},
		{Content: "\tvar a = 3", TrimmedContent: "var a = 3", HasASTInfo: true, StatementType: "var"},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)
	expected := "\tx := 1\n\n\tvar a = 3"

	if result != expected {
		t.Errorf("expected blank between different types, got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestEngineSuppressAfterOpenBrace(t *testing.T) {
	events := []LineEvent{
		{Content: "func main() {", TrimmedContent: "func main() {", HasASTInfo: true, StatementType: "func", IsScoped: true, IsTopLevel: true, IsStartLine: true, IsOpeningBrace: true},
		{Content: "\tif true {", TrimmedContent: "if true {", HasASTInfo: true, StatementType: "*ast.IfStmt", IsScoped: true, IsStartLine: true, IsOpeningBrace: true},
		{Content: "\t\tx := 1", TrimmedContent: "x := 1", HasASTInfo: true, StatementType: "*ast.AssignStmt"},
		{Content: "\t}", TrimmedContent: "}", IsClosingBrace: true},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)
	expected := "func main() {\n\tif true {\n\t\tx := 1\n\t}"

	if result != expected {
		t.Errorf("should not insert blank after open brace, got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestEngineSuppressBeforeCloseBrace(t *testing.T) {
	events := []LineEvent{
		{Content: "\tx := 1", TrimmedContent: "x := 1", HasASTInfo: true, StatementType: "*ast.AssignStmt", IsScoped: false},
		{Content: "}", TrimmedContent: "}", IsClosingBrace: true},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)
	expected := "\tx := 1\n}"

	if result != expected {
		t.Errorf("should not insert blank before close brace, got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestEngineSuppressCaseLabel(t *testing.T) {
	events := []LineEvent{
		{Content: "\tcase 1:", TrimmedContent: "case 1:", HasASTInfo: true, StatementType: "*ast.AssignStmt", IsCaseLabel: true, IsOpeningBrace: false},
		{Content: "\t\tfoo()", TrimmedContent: "foo()", HasASTInfo: true, StatementType: "*ast.ExprStmt"},
		{Content: "\tcase 2:", TrimmedContent: "case 2:", HasASTInfo: true, StatementType: "*ast.AssignStmt", IsCaseLabel: true},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)
	expected := "\tcase 1:\n\t\tfoo()\n\tcase 2:"

	if result != expected {
		t.Errorf("should not insert blank before case label, got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestEngineRawStringPassthrough(t *testing.T) {
	events := []LineEvent{
		{Content: "\tx := `", TrimmedContent: "x := `", HasASTInfo: true, StatementType: "*ast.AssignStmt"},
		{Content: "raw line 1", TrimmedContent: "raw line 1", InRawString: true},
		{Content: "", TrimmedContent: "", InRawString: true},
		{Content: "raw line 2`", TrimmedContent: "raw line 2`", InRawString: true},
		{Content: "\ty := 1", TrimmedContent: "y := 1", HasASTInfo: true, StatementType: "*ast.AssignStmt"},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)
	expected := "\tx := `\nraw line 1\n\nraw line 2`\n\ty := 1"

	if result != expected {
		t.Errorf("raw strings should pass through unchanged, got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestEngineTopLevelDifferentTypes(t *testing.T) {
	events := []LineEvent{
		{Content: "type Foo struct {", TrimmedContent: "type Foo struct {", HasASTInfo: true, StatementType: "type", IsTopLevel: true, IsScoped: true, IsStartLine: true, IsOpeningBrace: true},
		{Content: "\tX int", TrimmedContent: "X int"},
		{Content: "}", TrimmedContent: "}", IsClosingBrace: true, HasASTInfo: true, StatementType: "type", IsTopLevel: true, IsScoped: true},
		{Content: "var x = 1", TrimmedContent: "var x = 1", HasASTInfo: true, StatementType: "var", IsTopLevel: true, IsStartLine: true},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)
	expected := "type Foo struct {\n\tX int\n}\n\nvar x = 1"

	if result != expected {
		t.Errorf("expected blank between different top-level types, got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestEngineCommentLookAhead(t *testing.T) {
	events := []LineEvent{
		{Content: "\tx := 1", TrimmedContent: "x := 1", HasASTInfo: true, StatementType: "*ast.AssignStmt"},
		{Content: "\t// comment about if", TrimmedContent: "// comment about if", IsCommentOnly: true},
		{Content: "\tif true {", TrimmedContent: "if true {", HasASTInfo: true, StatementType: "*ast.IfStmt", IsScoped: true, IsStartLine: true, IsOpeningBrace: true},
		{Content: "\t\ty := 2", TrimmedContent: "y := 2", HasASTInfo: true, StatementType: "*ast.AssignStmt"},
		{Content: "\t}", TrimmedContent: "}", IsClosingBrace: true},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)
	expected := "\tx := 1\n\n\t// comment about if\n\tif true {\n\t\ty := 2\n\t}"

	if result != expected {
		t.Errorf("comment should trigger look-ahead blank, got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestEnginePackageDeclaration(t *testing.T) {
	events := []LineEvent{
		{Content: "package main", TrimmedContent: "package main", IsPackageDecl: true},
		{Content: "", TrimmedContent: "", IsBlank: true},
		{Content: "func main() {", TrimmedContent: "func main() {", HasASTInfo: true, StatementType: "func", IsTopLevel: true, IsScoped: true, IsStartLine: true, IsOpeningBrace: true},
		{Content: "}", TrimmedContent: "}", IsClosingBrace: true, HasASTInfo: true, StatementType: "func", IsTopLevel: true, IsScoped: true},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formatResult(formattingEngine, events)
	expected := "package main\n\nfunc main() {\n}"

	if result != expected {
		t.Errorf("package should separate from func, got:\n%s\nwant:\n%s", result, expected)
	}
}

func TestEngineFormatToString(t *testing.T) {
	events := []LineEvent{
		{Content: "package main", TrimmedContent: "package main", IsPackageDecl: true},
	}
	formattingEngine := &Engine{CommentMode: CommentsFollow}
	result := formattingEngine.FormatToString(events)

	if result != "package main\n" {
		t.Errorf("FormatToString should end with newline, got: %q", result)
	}
}

func TestEngineFindNextNonComment(t *testing.T) {
	events := []LineEvent{
		{Content: "x", TrimmedContent: "x"},
		{Content: "", TrimmedContent: "", IsBlank: true},
		{Content: "// comment", TrimmedContent: "// comment", IsCommentOnly: true},
		{Content: "y", TrimmedContent: "y"},
	}
	formattingEngine := &Engine{}
	index := formattingEngine.findNextNonComment(events, 1)

	if index != 3 {
		t.Errorf("expected index 3, got %d", index)
	}

	index = formattingEngine.findNextNonComment(events, 4)

	if index != -1 {
		t.Errorf("expected -1 when past end, got %d", index)
	}
}
