package engine

import "strings"

type LineEvent struct {
	Content        string
	TrimmedContent string
	StatementType  string
	IsTopLevel     bool
	IsScoped       bool
	IsStartLine    bool
	HasASTInfo     bool
	IsClosingBrace bool
	IsOpeningBrace bool
	IsCaseLabel    bool
	IsCommentOnly  bool
	IsBlank        bool
	InRawString    bool
	IsPackageDecl  bool
}

func NewLineEvent(content string) LineEvent {
	trimmed := strings.TrimSpace(content)

	return LineEvent{
		Content:        content,
		TrimmedContent: trimmed,
		IsBlank:        trimmed == "",
	}
}
