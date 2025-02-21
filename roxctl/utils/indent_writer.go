package utils

import (
	"io"
	"strings"
)

const tabWidth = 8

type indentAndWrapWriter struct {
	w      io.StringWriter
	width  int
	indent indents

	currentLine int
	written     int
}

var _ io.StringWriter = (*indentAndWrapWriter)(nil)

func makeWriter(w io.StringWriter, width int, indent ...int) *indentAndWrapWriter {
	return &indentAndWrapWriter{w: w, width: width, indent: indent}
}

//nolint:wrapcheck
func (w *indentAndWrapWriter) write0(s string) error {
	n, err := w.w.WriteString(s)
	w.currentLine += n
	w.written += n
	return err
}

func (w *indentAndWrapWriter) writePadding() error {
	w.currentLine = 0
	return w.write0(strings.Repeat(" ", w.indent.pop()))
}

//nolint:wrapcheck
func (w *indentAndWrapWriter) ln() error {
	if _, err := w.w.WriteString("\n"); err != nil {
		return err
	}
	return w.writePadding()
}

func (w *indentAndWrapWriter) WriteString(s string) (int, error) {
	w.written = 0
	if w.currentLine == 0 {
		if err := w.writePadding(); err != nil {
			return w.written, err
		}
	}
	for word := range words(s) {
		if word == "\n" {
			if err := w.ln(); err != nil {
				return w.written, err
			}
			continue
		}
		length := len(word)
		if word == "\n" {
			length = tabWidth
		}
		if w.currentLine+length > w.width {
			if err := w.ln(); err != nil {
				return w.written, err
			}
			if word == " " || word == "\t" {
				continue
			}
		}
		if err := w.write0(word); err != nil {
			return w.written, err
		}
	}
	return w.written, nil
}
