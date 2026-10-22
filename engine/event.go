package engine

type LineEvent struct {
	Content           string
	StatementType     string
	IsTopLevel        bool
	IsScoped          bool
	IsSingleLineScope bool
	HasASTInfo        bool
	IsClosingBrace    bool
	IsOpeningBrace    bool
	IsCaseLabel       bool
	IsContinuation    bool
	IsCommentOnly     bool
	IsBlank           bool
	InRawString       bool
}
