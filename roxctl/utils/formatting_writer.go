package utils

import (
	"io"
	"strings"
)

const tabWidth = 8

// formattingWriter implements StringWriter interface.
// It writes strings to the underlying writer indenting and wrapping the text.
type formattingWriter struct {
	raw    io.StringWriter
	width  int
	indent indents

	currentLine int
	written     int
	indentReset bool
}

var _ io.StringWriter = (*formattingWriter)(nil)

func makeFormattingWriter(w io.StringWriter, width int, indent ...int) *formattingWriter {
	return &formattingWriter{raw: w, width: width, indent: indent}
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
// required offset.
func (w *formattingWriter) writePadding() error {
	var err error
	padding := w.indent.pop()
	if padding < 0 {
		// Indent to tabulation.
		padding = -padding
		if padding > w.currentLine {
			padding -= w.currentLine
		} else {
			err = w.ln()
		}
	} else if w.currentLine != 0 && w.currentLine+padding > w.width {
		err = w.ln()
	}
	if err != nil {
		return err
	}
	return w.write(strings.Repeat(" ", padding))
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
	for word := range words(s) {
		if err != nil {
			break
		}
		if w.currentLine == 0 || w.indentReset {
			if err = w.writePadding(); err != nil {
				break
			}
			w.indentReset = false
		}
		if word == "\n" {
			err = w.ln()
			continue
		}
		length := len(word)
		if word == "\t" {
			length = tabWidth
		}
		if w.currentLine+length > w.width {
			if err = w.ln(); err != nil {
				break
			}
			if word == " " {
				continue
			}
			if err = w.writePadding(); err != nil {
				break
			}
		}
		err = w.write(word)
	}
	return w.written, err
}
