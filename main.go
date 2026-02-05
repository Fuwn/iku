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
	"sync"
)

var version = "dev"
var (
	writeFlag    = flag.Bool("w", false, "write result to (source) file instead of stdout")
	listFlag     = flag.Bool("l", false, "list files whose formatting differs from iku's")
	diffFlag     = flag.Bool("d", false, "display diffs instead of rewriting files")
	commentsFlag = flag.String("comments", "follow", "comment attachment mode: follow, precede, standalone")
	versionFlag  = flag.Bool("version", false, "print version")
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

	commentMode, err := parseCommentMode(*commentsFlag)

	if err != nil {
		fmt.Fprintf(os.Stderr, "iku: %v\n", err)
		os.Exit(2)
	}

	formatter := &Formatter{CommentMode: commentMode}

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
		switch fileInfo, err := os.Stat(argumentPath); {
		case err != nil:
			fmt.Fprintf(os.Stderr, "iku: %v\n", err)

			exitCode = 1
		case fileInfo.IsDir():
			if err := processDirectory(formatter, argumentPath, &exitCode); err != nil {
				fmt.Fprintf(os.Stderr, "iku: %v\n", err)

				exitCode = 1
			}
		default:
			if err := processFilePath(formatter, argumentPath, &exitCode); err != nil {
				fmt.Fprintf(os.Stderr, "iku: %v\n", err)

				exitCode = 1
			}
		}
	}

	os.Exit(exitCode)
}

func parseCommentMode(commentModeString string) (CommentMode, error) {
	switch strings.ToLower(commentModeString) {
	case "follow":
		return CommentsFollow, nil
	case "precede":
		return CommentsPrecede, nil
	case "standalone":
		return CommentsStandalone, nil
	default:
		return 0, fmt.Errorf("invalid comment mode: %q (use follow, precede, or standalone)", commentModeString)
	}
}

var supportedFileExtensions = map[string]bool{
	".go":  true,
	".js":  true,
	".ts":  true,
	".jsx": true,
	".tsx": true,
}

func processDirectory(formatter *Formatter, directoryPath string, exitCode *int) error {
	var sourceFilePaths []string

	err := filepath.WalkDir(directoryPath, func(currentPath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.IsDir() && supportedFileExtensions[filepath.Ext(currentPath)] {
			sourceFilePaths = append(sourceFilePaths, currentPath)
		}

		return nil
	})

	if err != nil {
		return err
	}

	var waitGroup sync.WaitGroup
	var mutex sync.Mutex

	semaphore := make(chan struct{}, runtime.NumCPU())

	for _, filePath := range sourceFilePaths {
		waitGroup.Add(1)

		go func(currentFilePath string) {
			defer waitGroup.Done()

			semaphore <- struct{}{}

			defer func() { <-semaphore }()

			if err := processFilePath(formatter, currentFilePath, exitCode); err != nil {
				mutex.Lock()
				fmt.Fprintf(os.Stderr, "iku: %v\n", err)

				*exitCode = 1

				mutex.Unlock()
			}
		}(filePath)
	}

	waitGroup.Wait()

	return nil
}

func processFilePath(formatter *Formatter, filePath string, _ *int) error {
	sourceFile, err := os.Open(filePath)

	if err != nil {
		return err
	}

	defer func() { _ = sourceFile.Close() }()

	var outputDestination io.Writer = os.Stdout

	if *writeFlag {
		outputDestination = nil
	}

	return processFile(formatter, filePath, sourceFile, outputDestination, true)
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

	if *listFlag {
		if !bytes.Equal(sourceContent, formattedResult) {
			fmt.Println(filename)
		}

		return nil
	}

	if *diffFlag {
		if !bytes.Equal(sourceContent, formattedResult) {
			diffOutput := unifiedDiff(filename, sourceContent, formattedResult)
			_, _ = os.Stdout.Write(diffOutput)
		}

		return nil
	}

	if *writeFlag && isFile {
		if !bytes.Equal(sourceContent, formattedResult) {
			return os.WriteFile(filename, formattedResult, 0644)
		}

		return nil
	}

	if outputWriter != nil {
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
