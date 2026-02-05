package main

import "github.com/Fuwn/iku/engine"

type CommentMode int

const (
	CommentsFollow CommentMode = iota
	CommentsPrecede
	CommentsStandalone
)

type Formatter struct {
	CommentMode CommentMode
}

type lineInformation struct {
	statementType string
	isTopLevel    bool
	isScoped      bool
	isStartLine   bool
}

func (f *Formatter) Format(source []byte) ([]byte, error) {
	adapter := &GoAdapter{}
	_, events, err := adapter.Analyze(source)

	if err != nil {
		return nil, err
	}

	formattingEngine := &engine.Engine{CommentMode: MapCommentMode(f.CommentMode)}

	return []byte(formattingEngine.FormatToString(events)), nil
}
