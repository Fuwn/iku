package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
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

	for _, path := range flag.Args() {
		switch info, err := os.Stat(path); {
		case err != nil:
			fmt.Fprintf(os.Stderr, "iku: %v\n", err)

			exitCode = 1
		case info.IsDir():
			if err := processDir(formatter, path, &exitCode); err != nil {
				fmt.Fprintf(os.Stderr, "iku: %v\n", err)

				exitCode = 1
			}
		default:
			if err := processFilePath(formatter, path, &exitCode); err != nil {
				fmt.Fprintf(os.Stderr, "iku: %v\n", err)

				exitCode = 1
			}
		}
	}

	os.Exit(exitCode)
}

func parseCommentMode(mode string) (CommentMode, error) {
	switch strings.ToLower(mode) {
	case "follow":
		return CommentsFollow, nil
	case "precede":
		return CommentsPrecede, nil
	case "standalone":
		return CommentsStandalone, nil
	default:
		return 0, fmt.Errorf("invalid comment mode: %q (use follow, precede, or standalone)", mode)
	}
}

func processDir(formatter *Formatter, directory string, exitCode *int) error {
	var files []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	var waitGroup sync.WaitGroup
	var mutex sync.Mutex

	semaphore := make(chan struct{}, runtime.NumCPU())

	for _, path := range files {
		waitGroup.Add(1)

		go func(filePath string) {
			defer waitGroup.Done()

			semaphore <- struct{}{}

			defer func() { <-semaphore }()

			if err := processFilePath(formatter, filePath, exitCode); err != nil {
				mutex.Lock()
				fmt.Fprintf(os.Stderr, "iku: %v\n", err)

				*exitCode = 1

				mutex.Unlock()
			}
		}(path)
	}

	waitGroup.Wait()

	return nil
}

func processFilePath(formatter *Formatter, path string, _ *int) error {
	file, err := os.Open(path)

	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }()

	var output io.Writer = os.Stdout

	if *writeFlag {
		output = nil
	}

	return processFile(formatter, path, file, output, true)
}

func processFile(formatter *Formatter, filename string, input io.Reader, outputWriter io.Writer, isFile bool) error {
	source, err := io.ReadAll(input)

	if err != nil {
		return fmt.Errorf("%s: %v", filename, err)
	}

	result, err := formatter.Format(source)

	if err != nil {
		return fmt.Errorf("%s: %v", filename, err)
	}

	if *listFlag {
		if !bytes.Equal(source, result) {
			fmt.Println(filename)
		}

		return nil
	}

	if *diffFlag {
		if !bytes.Equal(source, result) {
			difference := unifiedDiff(filename, source, result)
			_, _ = os.Stdout.Write(difference)
		}

		return nil
	}

	if *writeFlag && isFile {
		if !bytes.Equal(source, result) {
			return os.WriteFile(filename, result, 0644)
		}

		return nil
	}

	if outputWriter != nil {
		_, err = outputWriter.Write(result)

		return err
	}

	return nil
}

func unifiedDiff(filename string, original, formatted []byte) []byte {
	var buffer bytes.Buffer

	fmt.Fprintf(&buffer, "--- %s\n", filename)
	fmt.Fprintf(&buffer, "+++ %s\n", filename)

	originalLines := strings.Split(string(original), "\n")
	formattedLines := strings.Split(string(formatted), "\n")

	fmt.Fprintf(&buffer, "@@ -1,%d +1,%d @@\n", len(originalLines), len(formattedLines))

	for _, line := range originalLines {
		fmt.Fprintf(&buffer, "-%s\n", line)
	}

	for _, line := range formattedLines {
		fmt.Fprintf(&buffer, "+%s\n", line)
	}

	return buffer.Bytes()
}
