package utils

import (
	"io"
	"strings"
)

const tabWidth = 8

type formattingWriter struct {
	raw    io.StringWriter
	width  int
	indent indents

	currentLine int
	written     int
	indentReset bool
}

var _ io.StringWriter = (*formattingWriter)(nil)

func makeWriter(w io.StringWriter, width int, indent ...int) *formattingWriter {
	return &formattingWriter{raw: w, width: width, indent: indent}
}

//nolint:wrapcheck
func (w *formattingWriter) write0(s string) error {
	n, err := w.raw.WriteString(s)
	w.currentLine += n
	w.written += n
	return err
}

func (w *formattingWriter) writePadding() error {
	return w.write0(strings.Repeat(" ", w.indent.pop()))
}

//nolint:wrapcheck
func (w *formattingWriter) ln() error {
	if _, err := w.raw.WriteString("\n"); err != nil {
		return err
	}
	w.currentLine = 0
	return nil
}

func (w *formattingWriter) setIndent(indent ...int) {
	w.indentReset = true
	w.indent = indent
}

func (w *formattingWriter) WriteString(s string) (int, error) {
	w.written = 0
	for word := range words(s) {
		if w.currentLine == 0 || w.indentReset {
			if err := w.writePadding(); err != nil {
				return w.written, err
			}
			w.indentReset = false
		}
		if word == "\n" {
			if err := w.ln(); err != nil {
				return w.written, err
			}
			continue
		}
		length := len(word)
		if word == "\t" {
			length = tabWidth
		}
		if w.currentLine+length > w.width {
			if err := w.ln(); err != nil {
				return w.written, err
			}
			if word == " " {
				continue
			}
			if err := w.writePadding(); err != nil {
				return w.written, err
			}
		}
		if err := w.write0(word); err != nil {
			return w.written, err
		}
	}
	return w.written, nil
}
