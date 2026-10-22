package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func writeTestFiles(t *testing.T, directory string, contents map[string]string) {
	t.Helper()

	for relativePath, content := range contents {
		filePath := filepath.Join(directory, relativePath)

		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func runCommandLine(command *commandLine, argumentPaths ...string) (int, string, string) {
	var output, errorOutput bytes.Buffer

	command.formatter = &Formatter{}
	command.output = &output
	command.errorOutput = &errorOutput

	exitCode := command.processPaths(argumentPaths)

	return exitCode, output.String(), errorOutput.String()
}

func TestCommandLineListsDirectoryInWalkOrder(t *testing.T) {
	directory := t.TempDir()
	contents := map[string]string{}
	var expectedOutput strings.Builder

	for index := range 40 {
		relativePath := fmt.Sprintf("file%02d.go", index)

		contents[relativePath] = "package example\nvar first = 1\nfunc second() {}\n"

		fmt.Fprintln(&expectedOutput, filepath.Join(directory, relativePath))
	}

	writeTestFiles(t, directory, contents)

	exitCode, output, errorOutput := runCommandLine(&commandLine{list: true}, directory)

	if exitCode != 0 || errorOutput != "" {
		t.Fatalf("exit code %d, errors %q", exitCode, errorOutput)
	}

	if output != expectedOutput.String() {
		t.Errorf("got %q; want %q", output, expectedOutput.String())
	}
}

func TestCommandLineContinuesAfterErrors(t *testing.T) {
	directory := t.TempDir()
	formattedPath := filepath.Join(directory, "example.go")

	writeTestFiles(t, directory, map[string]string{"example.go": "package example\n"})

	exitCode, output, errorOutput := runCommandLine(&commandLine{}, filepath.Join(directory, "missing.go"), formattedPath)

	if exitCode != 1 || !strings.Contains(errorOutput, "missing.go") {
		t.Errorf("exit code %d, errors %q", exitCode, errorOutput)
	}

	if output != "package example\n" {
		t.Errorf("got %q; want the formatted file", output)
	}
}
