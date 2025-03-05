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

// writePadding is an internal method, that takes the next indent value and the
// writes the according number of spaces. If the indent value is negative, it is
// calculated as tabulation offset, i.e. the spaces are added to reach the
// required offset. Returns true if a new line has been written.
func (w *formattingWriter) writePadding() (bool, error) {
	var err error
	var ln bool
	padding := w.indent.popNotLast()
	if padding < 0 {
		// Indent to tabulation.
		padding = -padding
		if padding > w.currentLine {
			padding -= w.currentLine
		} else {
			err = w.ln()
			ln = true
		}
	} else if w.currentLine+padding > w.width {
		err = w.ln()
		ln = true
	}
	if err == nil {
		err = w.write(strings.Repeat(" ", padding))
	}
	return ln, err
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
	var err error
	tokenScanner := bufio.NewScanner(strings.NewReader(s))
	tokenScanner.Split(wordsAndDelimeters)
	for tokenScanner.Scan() {
		token := tokenScanner.Text()
		if err != nil {
			break
		}
		ln := false
		if w.currentLine == 0 || w.indentReset {
			if token == "\n" {
				w.indent.popNotLast()
			} else if ln, err = w.writePadding(); err != nil {
				break
			}
			w.indentReset = false
		}
		if token == "\n" {
			err = w.ln()
			continue
		}
		length := len(token)
		if token == "\t" {
			length = defaultTabWidth
		}
		if w.currentLine+length > w.width && !ln {
			if err = w.ln(); err != nil {
				break
			}
			if token == " " {
				continue
			}
			if _, err = w.writePadding(); err != nil {
				break
			}
		}
		err = w.write(token)
	}
	return w.written, err
}
