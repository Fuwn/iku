package main

import (
	"fmt"
	"slices"
	"strings"
)

const diffContextLines = 3

// Edit-path memory grows with the square of the edit count, so larger rewrites, such as a
// fully reindented file, become one replacement hunk instead.
const maximumTracedLineEdits = 1024

type lineEditKind byte

const (
	lineKept     lineEditKind = ' '
	lineRemoved  lineEditKind = '-'
	lineInserted lineEditKind = '+'
)

type lineEdit struct {
	kind          lineEditKind
	line          string
	originalLine  int
	formattedLine int
}

func unifiedDiff(filename string, originalSource, formattedSource []byte) []byte {
	edits := shortestLineEdits(splitLines(string(originalSource)), splitLines(string(formattedSource)))
	output := fmt.Appendf(nil, "--- %s\n+++ %s\n", filename, filename)

	for hunkStart := 0; hunkStart < len(edits); {
		if edits[hunkStart].kind == lineKept {
			hunkStart++

			continue
		}

		hunkEnd := hunkStart + 1
		keptRun := 0

		for hunkEnd < len(edits) && keptRun <= 2*diffContextLines {
			if edits[hunkEnd].kind == lineKept {
				keptRun++
			} else {
				keptRun = 0
			}

			hunkEnd++
		}

		hunkEnd -= max(0, keptRun-diffContextLines)
		output = appendHunk(output, edits[max(0, hunkStart-diffContextLines):hunkEnd])
		hunkStart = hunkEnd
	}

	return output
}

func appendHunk(output []byte, edits []lineEdit) []byte {
	originalCount := 0
	formattedCount := 0

	for _, edit := range edits {
		if edit.kind != lineInserted {
			originalCount++
		}

		if edit.kind != lineRemoved {
			formattedCount++
		}
	}

	output = fmt.Appendf(output, "@@ -%s +%s @@\n", hunkRange(edits[0].originalLine, originalCount), hunkRange(edits[0].formattedLine, formattedCount))

	for _, edit := range edits {
		output = append(output, byte(edit.kind))
		output = append(output, edit.line...)

		if !strings.HasSuffix(edit.line, "\n") {
			output = append(output, "\n\\ No newline at end of file\n"...)
		}
	}

	return output
}

// An empty range names the line before it, as in GNU diff.
func hunkRange(startIndex, lineCount int) string {
	if lineCount == 0 {
		return fmt.Sprintf("%d,0", startIndex)
	}

	return fmt.Sprintf("%d,%d", startIndex+1, lineCount)
}

// Each line keeps its newline so a missing final newline is itself a difference.
func splitLines(source string) []string {
	var lines []string

	for source != "" {
		lineEnd := strings.IndexByte(source, '\n') + 1

		if lineEnd == 0 {
			lineEnd = len(source)
		}

		lines = append(lines, source[:lineEnd])
		source = source[lineEnd:]
	}

	return lines
}

// Myers' greedy algorithm keeps the furthest original position on each diagonal, with
// diagonal k holding positions where originalIndex-formattedIndex == k. Each depth's
// frontier is saved so the path can be recovered; memory grows with the square of the edit
// count rather than with file length.
func shortestLineEdits(originalLines, formattedLines []string) []lineEdit {
	originalCount := len(originalLines)
	formattedCount := len(formattedLines)
	diagonalOffset := originalCount + formattedCount + 1
	furthestOnDiagonal := make([]int, 2*diagonalOffset+1)
	var frontiers [][]int

	for depth := 0; ; depth++ {
		if depth > maximumTracedLineEdits {
			return replacementLineEdits(originalLines, formattedLines)
		}

		frontiers = append(frontiers, slices.Clone(furthestOnDiagonal[diagonalOffset-depth-1:diagonalOffset+depth+2]))

		for diagonal := -depth; diagonal <= depth; diagonal += 2 {
			originalIndex := furthestOnDiagonal[diagonalOffset+diagonal-1] + 1

			if diagonal == -depth || (diagonal != depth && furthestOnDiagonal[diagonalOffset+diagonal-1] < furthestOnDiagonal[diagonalOffset+diagonal+1]) {
				originalIndex = furthestOnDiagonal[diagonalOffset+diagonal+1]
			}

			formattedIndex := originalIndex - diagonal

			for originalIndex < originalCount && formattedIndex < formattedCount && originalLines[originalIndex] == formattedLines[formattedIndex] {
				originalIndex++
				formattedIndex++
			}

			furthestOnDiagonal[diagonalOffset+diagonal] = originalIndex

			if originalIndex >= originalCount && formattedIndex >= formattedCount {
				return traceLineEdits(frontiers, originalLines, formattedLines)
			}
		}
	}
}

func traceLineEdits(frontiers [][]int, originalLines, formattedLines []string) []lineEdit {
	originalIndex := len(originalLines)
	formattedIndex := len(formattedLines)
	var edits []lineEdit

	for depth := len(frontiers) - 1; depth > 0; depth-- {
		frontier := frontiers[depth]
		furthestBefore := func(diagonal int) int { return frontier[diagonal+depth+1] }
		diagonal := originalIndex - formattedIndex
		previousDiagonal := diagonal - 1

		if diagonal == -depth || (diagonal != depth && furthestBefore(diagonal-1) < furthestBefore(diagonal+1)) {
			previousDiagonal = diagonal + 1
		}

		previousOriginalIndex := furthestBefore(previousDiagonal)
		previousFormattedIndex := previousOriginalIndex - previousDiagonal

		for originalIndex > previousOriginalIndex && formattedIndex > previousFormattedIndex {
			originalIndex--
			formattedIndex--
			edits = append(edits, lineEdit{lineKept, formattedLines[formattedIndex], originalIndex, formattedIndex})
		}

		if originalIndex == previousOriginalIndex {
			formattedIndex--
			edits = append(edits, lineEdit{lineInserted, formattedLines[formattedIndex], originalIndex, formattedIndex})
		} else {
			originalIndex--
			edits = append(edits, lineEdit{lineRemoved, originalLines[originalIndex], originalIndex, formattedIndex})
		}
	}

	for originalIndex > 0 {
		originalIndex--
		formattedIndex--
		edits = append(edits, lineEdit{lineKept, formattedLines[formattedIndex], originalIndex, formattedIndex})
	}

	slices.Reverse(edits)

	return edits
}

func replacementLineEdits(originalLines, formattedLines []string) []lineEdit {
	edits := make([]lineEdit, 0, len(originalLines)+len(formattedLines))

	for originalIndex, line := range originalLines {
		edits = append(edits, lineEdit{lineRemoved, line, originalIndex, 0})
	}

	for formattedIndex, line := range formattedLines {
		edits = append(edits, lineEdit{lineInserted, line, len(originalLines), formattedIndex})
	}

	return edits
}
