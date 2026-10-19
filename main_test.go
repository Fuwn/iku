package main

import "testing"

func TestUnifiedDiff(t *testing.T) {
	cases := []struct {
		name      string
		original  string
		formatted string
		expected  string
	}{
		{"EmptyOriginal", "", "\n", "--- example.go\n+++ example.go\n@@ -1,1 +1,2 @@\n-\n+\n+\n"},
		{"NoFinalNewline", "old", "new", "--- example.go\n+++ example.go\n@@ -1,1 +1,1 @@\n-old\n+new\n"},
		{"FinalNewline", "old\n", "new\n", "--- example.go\n+++ example.go\n@@ -1,2 +1,2 @@\n-old\n-\n+new\n+\n"},
		{"CarriageReturns", "old\r\n", "new\r\n", "--- example.go\n+++ example.go\n@@ -1,2 +1,2 @@\n-old\r\n-\n+new\r\n+\n"},
		{"ArbitraryBytes", "\xff\n", "\xfe\n", "--- example.go\n+++ example.go\n@@ -1,2 +1,2 @@\n-\xff\n-\n+\xfe\n+\n"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := unifiedDiff("example.go", []byte(testCase.original), []byte(testCase.formatted))

			if string(result) != testCase.expected {
				t.Errorf("got %q; want %q", result, testCase.expected)
			}
		})
	}
}
