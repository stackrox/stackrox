package utils

import (
	"bufio"
	"io"
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
func (w *formattingWriter) write(s string) error {
	n, err := w.raw.WriteString(s)
	w.currentLine += n
	w.written += n
	return err //nolint:wrapcheck
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

// ln is an internal method that writes a new line to the underlying writer.
func (w *formattingWriter) ln() error {
	_, err := w.raw.WriteString("\n")
	w.currentLine = 0
	return err //nolint:wrapcheck
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
	w.written = 0
	tokenScanner := bufio.NewScanner(strings.NewReader(s))
	tokenScanner.Split(wordsAndDelimeters)
	var err error
	for err == nil && tokenScanner.Scan() {
		token := tokenScanner.Text()
		length := len(token)
		switch token {
		case "\t":
			length = defaultTabWidth
		case "\n":
			if w.currentLine == 0 {
				w.indent.popNotLast()
				w.indentReset = false
			}
			err = w.ln()
			continue
		}
		ln, padding := w.computePadding()
		if ln || w.currentLine+padding+length > w.width {
			if err = w.ln(); err != nil || token == " " {
				continue
			}
			_, padding = w.computePadding()
		}

		if err = w.write(strings.Repeat(" ", padding)); err == nil {
			err = w.write(token)
		}
	}
	return w.written, err
}
