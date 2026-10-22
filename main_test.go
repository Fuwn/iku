package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestCommandLineListsAndWrites(t *testing.T) {
	directory := t.TempDir()
	filePath := filepath.Join(directory, "example.go")

	writeTestFiles(t, directory, map[string]string{"example.go": "package example\nvar first = 1\nfunc second() {}\n"})

	exitCode, output, errorOutput := runCommandLine(&commandLine{list: true, write: true}, filePath)

	if exitCode != 0 || errorOutput != "" || output != filePath+"\n" {
		t.Fatalf("exit code %d, output %q, errors %q", exitCode, output, errorOutput)
	}

	written, err := os.ReadFile(filePath)

	if err != nil {
		t.Fatal(err)
	}

	if expected := "package example\n\nvar first = 1\n\nfunc second() {}\n"; string(written) != expected {
		t.Errorf("got %q; want %q", written, expected)
	}
}

func TestCommandLineSkipsHiddenAndDependencyDirectories(t *testing.T) {
	directory := t.TempDir()
	unformatted := "const first = 1;\nrun();\n"

	writeTestFiles(t, directory, map[string]string{
		".git/hooks/example.js":           unformatted,
		"node_modules/package/example.js": unformatted,
		"source/example.js":               unformatted,
	})

	exitCode, output, errorOutput := runCommandLine(&commandLine{list: true}, directory)

	if expected := filepath.Join(directory, "source", "example.js") + "\n"; exitCode != 0 || errorOutput != "" || output != expected {
		t.Fatalf("exit code %d, output %q, errors %q; want output %q", exitCode, output, errorOutput, expected)
	}
}
