package utils

import (
	"bufio"
	"io"
	"iter"
	"strings"
)

const defaultTabWidth = 8

// formattingWriter implements StringWriter interface.
// It writes strings to the underlying writer indenting and wrapping the text.
type formattingWriter struct {
	raw      io.StringWriter
	width    int
	indent   indents
	tabWidth int

	currentLine int
	written     int
	indentReset bool
}

var _ io.StringWriter = (*formattingWriter)(nil)

func makeFormattingWriter(w io.StringWriter, width int, tabWidth int, indent ...int) *formattingWriter {
	return &formattingWriter{raw: w, width: width, indent: indent, tabWidth: tabWidth}
}

// write is an internal method that writes the string to the underlying writer.
func (w *formattingWriter) write(s string) (int, error) {
	n, err := w.raw.WriteString(s)
	if s == "\n" {
		w.currentLine = 0
	} else {
		w.currentLine += n
	}
	return n, err //nolint:wrapcheck
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
// Implements the StringWriter interface.
func (w *formattingWriter) WriteString(s string) (int, error) {
	written := 0
	for t := range w.tokens(s) {
		n, err := w.write(t)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

// tokens is an internal iterator that issues tokens with paddings and wrapping.
func (w *formattingWriter) tokens(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		tokenScanner := bufio.NewScanner(strings.NewReader(s))
	tokenLoop:
		for tokenScanner.Split(wordsAndDelimeters); tokenScanner.Scan(); {
			token := tokenScanner.Text()
			tokenLength := len(token)
			switch token {
			case "\t":
				tokenLength = defaultTabWidth
			case "\n":
				if w.currentLine == 0 {
					w.indent.popNotLast()
					w.indentReset = false
				}
				if !yield("\n") {
					break tokenLoop
				}
				continue
			}
			ln, padding := w.computePadding()
			if ln || w.currentLine+padding+tokenLength > w.width {
				if !yield("\n") {
					break
				}
				if token == " " {
					// Ignore the space that caused wrapping.
					continue
				}
				_, padding = w.computePadding()
			}
			if (padding > 0 && !yield(strings.Repeat(" ", padding))) ||
				!yield(token) {
				break
			}
		}
	}
}
