package utils

import (
	"bufio"
	"io"
	"iter"
	"strings"
)

const defaultTabWidth = 8

// formattingWriter writes strings to the underlying writer indenting and
// wrapping the text.
type formattingWriter struct {
	raw      io.StringWriter
	width    int
	indent   indents
	tabWidth int

	currentLine int
	indentReset bool
}

func makeFormattingWriter(w io.StringWriter, width int, tabWidth int, indent ...int) *formattingWriter {
	return &formattingWriter{raw: w, width: width, indent: indent, tabWidth: tabWidth}
}

// computePadding is an internal method, that takes the next indent value and
// computes the required number of spaces. If the indent value is negative, it
// is calculated as tabulation offset, i.e. the spaces are added to reach the
// required offset. Returns true if a new line needs to been written.
func (w *formattingWriter) computePadding() (bool, int) {
	if w.currentLine != 0 && !w.indentReset {
		return false, 0
	}
	w.indentReset = false
	padding := w.indent.popNotLast()
	if padding < 0 {
		// Indent to tabulation.
		padding = -padding
		if padding > w.currentLine {
			padding -= w.currentLine
		} else {
			return true, 0
		}
	} else if w.currentLine+padding > w.width {
		return true, 0
	}
	return false, padding
}

// SetIndent updates the indentation for the following writings.
func (w *formattingWriter) SetIndent(indent ...int) {
	w.indentReset = true
	w.indent = indent
}

// WriteString writes the string to the underlying writer, with the configured
// indentation and wrapping.
func (w *formattingWriter) WriteString(s string) error {
	for token := range w.tokens(strings.NewReader(s)) {
		if _, err := w.raw.WriteString(token); err != nil {
			return err //nolint:wrapcheck
		}
	}
	return nil
}

// tokens is an internal iterator that issues tokens with paddings and wrapping.
func (w *formattingWriter) tokens(r io.Reader) iter.Seq[string] {
	return func(yield func(string) bool) {
		tokenScanner := bufio.NewScanner(r)
		tokenScanner.Split(wordsAndDelimeters)
		for tokenScanner.Scan() {
			token := tokenScanner.Text()
			tokenLength := len(token)
			switch token {
			case "\t":
				tokenLength = w.tabWidth
			case "\n":
				if w.currentLine == 0 {
					w.indent.popNotLast()
					w.indentReset = false
				}
				w.currentLine = 0
				if !yield("\n") {
					return
				}
				continue
			}
			ln, padding := w.computePadding()
			// Wrapping condition:
			if ln || w.currentLine+padding+tokenLength > w.width {
				w.currentLine = 0
				if !yield("\n") {
					return
				}
				if token == " " {
					// Ignore the space that caused wrapping.
					continue
				}
				// Re-compute padding after wrapping.
				_, padding = w.computePadding()
			}
			w.currentLine += padding + tokenLength
			if (padding > 0 && !yield(strings.Repeat(" ", padding))) ||
				!yield(token) {
				return
			}
		}
	}
}
