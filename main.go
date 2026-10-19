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
	"strings"
)

var version = "dev"
var (
	writeFlag   = flag.Bool("w", false, "write result to (source) file instead of stdout")
	listFlag    = flag.Bool("l", false, "list files whose formatting differs from iku's")
	diffFlag    = flag.Bool("d", false, "display diffs instead of rewriting files")
	versionFlag = flag.Bool("version", false, "print version")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: iku [flags] [path ...]\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s (%s)\n", version, runtime.Version())
		os.Exit(0)
	}

	configuration := loadConfiguration()
	commentMode, validationError := configuration.commentMode()

	if validationError != nil {
		fmt.Fprintf(os.Stderr, "iku: %v\n", validationError)
		os.Exit(2)
	}

	formatter := &Formatter{CommentMode: commentMode, Configuration: configuration}

	if flag.NArg() == 0 {
		if *writeFlag {
			fmt.Fprintln(os.Stderr, "iku: cannot use -w with standard input")
			os.Exit(2)
		}

		if err := processFile(formatter, "<stdin>", os.Stdin, os.Stdout, false); err != nil {
			fmt.Fprintf(os.Stderr, "iku: %v\n", err)
			os.Exit(1)
		}

		return
	}

	exitCode := 0

	for _, argumentPath := range flag.Args() {
		fileInformation, processingError := os.Stat(argumentPath)

		if processingError == nil {
			if fileInformation.IsDir() {
				processingError = processDirectory(formatter, argumentPath, &exitCode)
			} else {
				processingError = processFilePath(formatter, argumentPath)
			}
		}

		if processingError != nil {
			fmt.Fprintf(os.Stderr, "iku: %v\n", processingError)

			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

func processDirectory(formatter *Formatter, directoryPath string, exitCode *int) error {
	var sourceFilePaths []string
	err := filepath.WalkDir(directoryPath, func(currentPath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		fileExtension := filepath.Ext(currentPath)

		if !dirEntry.IsDir() && (fileExtension == ".go" || isEcmaScriptExtension(fileExtension)) {
			sourceFilePaths = append(sourceFilePaths, currentPath)
		}

		return nil
	})

	if err != nil {
		return err
	}

	semaphore := make(chan struct{}, runtime.NumCPU())
	fileResults := make(chan error, runtime.NumCPU())

	for _, filePath := range sourceFilePaths {
		go func() {
			semaphore <- struct{}{}

			defer func() { <-semaphore }()

			fileResults <- processFilePath(formatter, filePath)
		}()
	}

	for range sourceFilePaths {
		if err := <-fileResults; err != nil {
			fmt.Fprintf(os.Stderr, "iku: %v\n", err)

			*exitCode = 1
		}
	}

	return nil
}

func processFilePath(formatter *Formatter, filePath string) error {
	sourceFile, err := os.Open(filePath)

	if err != nil {
		return err
	}

	defer func() { _ = sourceFile.Close() }()

	return processFile(formatter, filePath, sourceFile, os.Stdout, true)
}

func processFile(formatter *Formatter, filename string, inputReader io.Reader, outputWriter io.Writer, isFile bool) error {
	sourceContent, err := io.ReadAll(inputReader)

	if err != nil {
		return fmt.Errorf("%s: %v", filename, err)
	}

	formattedResult, err := formatter.Format(sourceContent, filename)

	if err != nil {
		return fmt.Errorf("%s: %v", filename, err)
	}

	writeToFile := *writeFlag && isFile

	if (*listFlag || *diffFlag || writeToFile) && bytes.Equal(sourceContent, formattedResult) {
		return nil
	}

	switch {
	case *listFlag:
		fmt.Println(filename)
	case *diffFlag:
		diffOutput := unifiedDiff(filename, sourceContent, formattedResult)

		_, _ = os.Stdout.Write(diffOutput)
	case writeToFile:
		return os.WriteFile(filename, formattedResult, 0644)
	case outputWriter != nil:
		_, err = outputWriter.Write(formattedResult)

		return err
	}

	return nil
}

func unifiedDiff(filename string, originalSource, formattedSource []byte) []byte {
	var outputBuffer bytes.Buffer

	fmt.Fprintf(&outputBuffer, "--- %s\n", filename)
	fmt.Fprintf(&outputBuffer, "+++ %s\n", filename)

	originalSourceLines := strings.Split(string(originalSource), "\n")
	formattedSourceLines := strings.Split(string(formattedSource), "\n")

	fmt.Fprintf(&outputBuffer, "@@ -1,%d +1,%d @@\n", len(originalSourceLines), len(formattedSourceLines))

	for _, currentLine := range originalSourceLines {
		fmt.Fprintf(&outputBuffer, "-%s\n", currentLine)
	}

	for _, currentLine := range formattedSourceLines {
		fmt.Fprintf(&outputBuffer, "+%s\n", currentLine)
	}

	return outputBuffer.Bytes()
}
