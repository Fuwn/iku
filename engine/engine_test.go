package engine

import "testing"

func TestEngineFormat(t *testing.T) {
	cases := []struct {
		name     string
		events   []LineEvent
		expected string
	}{
		{
			name: "CollapsesBlanks",
			events: []LineEvent{
				{Content: "\tx := 1", HasRole: true, StatementType: "binding"},
				{IsBlank: true},
				{IsBlank: true},
				{Content: "\ty := 2", HasRole: true, StatementType: "binding"},
			},
			expected: "\tx := 1\n\ty := 2\n",
		},
		{
			name: "ScopeBoundary",
			events: []LineEvent{
				{Content: "\tx := 1", HasRole: true, StatementType: "binding"},
				{Content: "\tif x > 0 {", HasRole: true, StatementType: "if", IsScoped: true, IsOpeningBrace: true},
				{Content: "\t\ty := 2", HasRole: true, StatementType: "binding"},
				{Content: "\t}", HasRole: true, StatementType: "if", IsScoped: true, IsClosingBrace: true},
				{Content: "\tz := 3", HasRole: true, StatementType: "binding"},
			},
			expected: "\tx := 1\n\n\tif x > 0 {\n\t\ty := 2\n\t}\n\n\tz := 3\n",
		},
		{
			name: "StatementTypeTransition",
			events: []LineEvent{
				{Content: "\tx = 1", HasRole: true, StatementType: "assignment"},
				{Content: "\tvar a = 3", HasRole: true, StatementType: "binding"},
			},
			expected: "\tx = 1\n\n\tvar a = 3\n",
		},
		{
			name: "NoBlankAfterOpeningBrace",
			events: []LineEvent{
				{Content: "func main() {", HasRole: true, StatementType: "function", IsScoped: true, IsTopLevel: true, IsOpeningBrace: true},
				{Content: "\tif true {", HasRole: true, StatementType: "if", IsScoped: true, IsOpeningBrace: true},
				{Content: "\t\tx := 1", HasRole: true, StatementType: "binding"},
				{Content: "\t}", IsClosingBrace: true},
			},
			expected: "func main() {\n\tif true {\n\t\tx := 1\n\t}\n",
		},
		{
			name: "NoBlankBeforeClosingBrace",
			events: []LineEvent{
				{Content: "\tx := 1", HasRole: true, StatementType: "binding"},
				{Content: "}", IsClosingBrace: true},
			},
			expected: "\tx := 1\n}\n",
		},
		{
			name: "NoBlankAroundCaseLabels",
			events: []LineEvent{
				{Content: "\tcase 1:", HasRole: true, StatementType: "case", IsCaseLabel: true},
				{Content: "\t\tfoo()", HasRole: true, StatementType: "call"},
				{Content: "\tcase 2:", HasRole: true, StatementType: "case", IsCaseLabel: true},
			},
			expected: "\tcase 1:\n\t\tfoo()\n\tcase 2:\n",
		},
		{
			name: "VerbatimLinesPassThrough",
			events: []LineEvent{
				{Content: "\tx := `", HasRole: true, StatementType: "binding"},
				{Content: "raw line 1", IsVerbatim: true},
				{Content: "", IsBlank: true, IsVerbatim: true},
				{Content: "raw line 2`", IsVerbatim: true},
				{Content: "\ty := 1", HasRole: true, StatementType: "binding"},
			},
			expected: "\tx := `\nraw line 1\n\nraw line 2`\n\ty := 1\n",
		},
		{
			name: "TopLevelKindChange",
			events: []LineEvent{
				{Content: "type Foo struct {", HasRole: true, StatementType: "type", IsTopLevel: true, IsScoped: true, IsOpeningBrace: true},
				{Content: "\tX int"},
				{Content: "}", HasRole: true, StatementType: "type", IsTopLevel: true, IsScoped: true, IsClosingBrace: true},
				{Content: "var x = 1", HasRole: true, StatementType: "binding", IsTopLevel: true},
			},
			expected: "type Foo struct {\n\tX int\n}\n\nvar x = 1\n",
		},
		{
			name: "CommentAttachesToScope",
			events: []LineEvent{
				{Content: "\tx := 1", HasRole: true, StatementType: "binding"},
				{Content: "\t// comment about if", IsCommentOnly: true},
				{Content: "\tif true {", HasRole: true, StatementType: "if", IsScoped: true, IsOpeningBrace: true},
				{Content: "\t\ty := 2", HasRole: true, StatementType: "binding"},
				{Content: "\t}", IsClosingBrace: true},
			},
			expected: "\tx := 1\n\n\t// comment about if\n\tif true {\n\t\ty := 2\n\t}\n",
		},
		{
			name: "PackageClause",
			events: []LineEvent{
				{Content: "package main", StatementType: "package"},
				{IsBlank: true},
				{Content: "var x = 1", HasRole: true, StatementType: "binding", IsTopLevel: true},
			},
			expected: "package main\n\nvar x = 1\n",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := string((&Engine{CommentMode: CommentsFollow}).Format(testCase.events))

			if result != testCase.expected {
				t.Errorf("got %q; want %q", result, testCase.expected)
			}
		})
	}
}

func TestEngineCommentModes(t *testing.T) {
	events := []LineEvent{
		{Content: "first()", StatementType: "call", HasRole: true},
		{Content: "// Preserve the required order.", IsCommentOnly: true},
		{Content: "second()", StatementType: "call", HasRole: true},
	}
	cases := []struct {
		mode     CommentMode
		expected string
	}{
		{CommentsFollow, "first()\n\n// Preserve the required order.\nsecond()\n"},
		{CommentsPrecede, "first()\n// Preserve the required order.\n\nsecond()\n"},
		{CommentsStandalone, "first()\n\n// Preserve the required order.\n\nsecond()\n"},
	}

	for _, testCase := range cases {
		result := string((&Engine{CommentMode: testCase.mode}).Format(events))

		if result != testCase.expected {
			t.Errorf("mode %d: got %q; want %q", testCase.mode, result, testCase.expected)
		}
	}
}
