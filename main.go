package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
)

var version = "dev"

type commandLine struct {
	formatter   *Formatter
	write       bool
	list        bool
	diff        bool
	output      io.Writer
	errorOutput io.Writer
}

type fileResult struct {
	output []byte
	err    error
}

func main() {
	writeFlag := flag.Bool("w", false, "write result to (source) file instead of stdout")
	listFlag := flag.Bool("l", false, "list files whose formatting differs from iku's")
	diffFlag := flag.Bool("d", false, "display diffs instead of rewriting files")
	versionFlag := flag.Bool("version", false, "print version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: iku [flags] [path ...]\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s (%s)\n", version, runtime.Version())
		os.Exit(0)
	}

	formattingEngine, validationError := loadConfiguration().formattingEngine()

	if validationError != nil {
		fmt.Fprintf(os.Stderr, "iku: %v\n", validationError)
		os.Exit(2)
	}

	command := &commandLine{
		formatter:   &Formatter{Engine: formattingEngine},
		write:       *writeFlag,
		list:        *listFlag,
		diff:        *diffFlag,
		output:      os.Stdout,
		errorOutput: os.Stderr,
	}

	if flag.NArg() == 0 {
		if command.write {
			fmt.Fprintln(os.Stderr, "iku: cannot use -w with standard input")
			os.Exit(2)
		}

		os.Exit(command.processStandardInput(os.Stdin))
	}

	os.Exit(command.processPaths(flag.Args()))
}

func (command *commandLine) processStandardInput(input io.Reader) int {
	source, err := io.ReadAll(input)

	if err != nil {
		return command.report(fileResult{err: err})
	}

	return command.report(command.formatSource("<stdin>", source))
}

func (command *commandLine) processPaths(argumentPaths []string) int {
	exitCode := 0

	for _, argumentPath := range argumentPaths {
		exitCode = max(exitCode, command.processPath(argumentPath))
	}

	return exitCode
}

func (command *commandLine) processPath(argumentPath string) int {
	fileInformation, err := os.Stat(argumentPath)

	if err != nil {
		return command.report(fileResult{err: err})
	}

	if !fileInformation.IsDir() {
		return command.report(command.formatFile(argumentPath))
	}

	sourceFilePaths, err := findSourceFiles(argumentPath)

	if err != nil {
		return command.report(fileResult{err: err})
	}

	// Workers finish in any order; reporting waits on each file's result in walk order.
	workerSlots := make(chan struct{}, runtime.NumCPU())
	pendingResults := make([]chan fileResult, len(sourceFilePaths))

	for index, filePath := range sourceFilePaths {
		pendingResults[index] = make(chan fileResult, 1)

		go func() {
			workerSlots <- struct{}{}

			defer func() { <-workerSlots }()

			pendingResults[index] <- command.formatFile(filePath)
		}()
	}

	exitCode := 0

	for _, pendingResult := range pendingResults {
		exitCode = max(exitCode, command.report(<-pendingResult))
	}

	return exitCode
}

func findSourceFiles(directoryPath string) ([]string, error) {
	var sourceFilePaths []string
	err := filepath.WalkDir(directoryPath, func(currentPath string, directoryEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		fileExtension := filepath.Ext(currentPath)

		if !directoryEntry.IsDir() && (fileExtension == ".go" || isEcmaScriptExtension(fileExtension)) {
			sourceFilePaths = append(sourceFilePaths, currentPath)
		}

		return nil
	})

	return sourceFilePaths, err
}

func (command *commandLine) formatFile(filePath string) fileResult {
	source, err := os.ReadFile(filePath)

	if err != nil {
		return fileResult{err: err}
	}

	return command.formatSource(filePath, source)
}

func (command *commandLine) formatSource(filename string, source []byte) fileResult {
	formatted, err := command.formatter.Format(source, filename)

	if err != nil {
		return fileResult{err: fmt.Errorf("%s: %v", filename, err)}
	}

	if (command.list || command.diff || command.write) && bytes.Equal(source, formatted) {
		return fileResult{}
	}

	switch {
	case command.list:
		return fileResult{output: []byte(filename + "\n")}
	case command.diff:
		return fileResult{output: unifiedDiff(filename, source, formatted)}
	case command.write:
		return fileResult{err: os.WriteFile(filename, formatted, 0644)}
	default:
		return fileResult{output: formatted}
	}
}

func (command *commandLine) report(result fileResult) int {
	if _, err := command.output.Write(result.output); err != nil && result.err == nil {
		result.err = err
	}

	if result.err == nil {
		return 0
	}

	fmt.Fprintf(command.errorOutput, "iku: %v\n", result.err)

	return 1
}

func unifiedDiff(filename string, originalSource, formattedSource []byte) []byte {
	originalLineCount := bytes.Count(originalSource, []byte("\n")) + 1
	formattedLineCount := bytes.Count(formattedSource, []byte("\n")) + 1
	output := fmt.Appendf(nil, "--- %s\n+++ %s\n@@ -1,%d +1,%d @@\n", filename, filename, originalLineCount, formattedLineCount)

	// Each source gains a prefix on every line and a trailing newline.
	output = slices.Grow(output, len(originalSource)+len(formattedSource)+originalLineCount+formattedLineCount+2)

	writeLines := func(prefix byte, source []byte) {
		for currentLine := range bytes.SplitSeq(source, []byte("\n")) {
			output = append(output, prefix)
			output = append(output, currentLine...)
			output = append(output, '\n')
		}
	}

	writeLines('-', originalSource)
	writeLines('+', formattedSource)

	return output
}
