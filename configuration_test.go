package main

import "testing"

func TestParseFormattingEngineRejectsInvalidConfiguration(t *testing.T) {
	for _, configuration := range []string{
		`{"comment_mode": "precede",, }`,
		`{"group_single_line_function": false}`,
		`{"comment_mode": "above"}`,
		`{"group_single_line_functions": "no"}`,
	} {
		if _, err := parseFormattingEngine([]byte(configuration)); err == nil {
			t.Errorf("configuration %s: expected an error", configuration)
		}
	}
}
