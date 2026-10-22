package main

import (
	"strings"
	"testing"
)

func numberedLines(first, last int, insertBlankAfter ...int) string {
	var lines strings.Builder

	for lineNumber := first; lineNumber <= last; lineNumber++ {
		lines.WriteString(strings.Repeat("x", lineNumber) + "\n")

		for _, blankAfter := range insertBlankAfter {
			if blankAfter == lineNumber {
				lines.WriteString("\n")
			}
		}
	}

	return lines.String()
}

func TestUnifiedDiff(t *testing.T) {
	cases := []struct {
		name      string
		original  string
		formatted string
		hunks     string
	}{
		{"EmptyOriginal", "", "\n", "@@ -0,0 +1,1 @@\n+\n"},
		{"NoFinalNewline", "old", "new", "@@ -1,1 +1,1 @@\n-old\n\\ No newline at end of file\n+new\n\\ No newline at end of file\n"},
		{"FinalNewline", "old\n", "new\n", "@@ -1,1 +1,1 @@\n-old\n+new\n"},
		{"AddedFinalNewline", "old", "old\n", "@@ -1,1 +1,1 @@\n-old\n\\ No newline at end of file\n+old\n"},
		{"CarriageReturns", "old\r\n", "new\r\n", "@@ -1,1 +1,1 @@\n-old\r\n+new\r\n"},
		{"ArbitraryBytes", "\xff\n", "\xfe\n", "@@ -1,1 +1,1 @@\n-\xff\n+\xfe\n"},
		{
			"SeparateHunks",
			numberedLines(1, 20),
			numberedLines(1, 20, 2, 18),
			"@@ -1,5 +1,6 @@\n x\n xx\n+\n xxx\n xxxx\n xxxxx\n" +
				"@@ -16,5 +17,6 @@\n " + strings.Repeat("x", 16) + "\n " + strings.Repeat("x", 17) + "\n " + strings.Repeat("x", 18) + "\n+\n " + strings.Repeat("x", 19) + "\n " + strings.Repeat("x", 20) + "\n",
		},
		{
			"MergedHunks",
			numberedLines(1, 12),
			numberedLines(1, 12, 2, 8),
			"@@ -1,11 +1,13 @@\n x\n xx\n+\n xxx\n xxxx\n xxxxx\n xxxxxx\n xxxxxxx\n xxxxxxxx\n+\n xxxxxxxxx\n xxxxxxxxxx\n xxxxxxxxxxx\n",
		},
		{
			"LargeRewriteReplacesWholeFile",
			strings.Repeat("a\n", maximumTracedLineEdits),
			strings.Repeat("b\n", maximumTracedLineEdits),
			"@@ -1,1024 +1,1024 @@\n" + strings.Repeat("-a\n", maximumTracedLineEdits) + strings.Repeat("+b\n", maximumTracedLineEdits),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := string(unifiedDiff("example.go", []byte(testCase.original), []byte(testCase.formatted)))
			expected := "--- example.go\n+++ example.go\n" + testCase.hunks

			if result != expected {
				t.Errorf("got %q; want %q", result, expected)
			}
		})
	}
}
