package utils

import (
	"io"

	"github.com/spf13/cobra"
)

// cobraWriter implements StringWriter using cobra.command.Print[Ln] functions.
type cobraWriter struct {
	c *cobra.Command
}

var _ io.StringWriter = (*cobraWriter)(nil)

func (cw *cobraWriter) WriteString(s string) (int, error) {
	cw.c.Print(s)
	return len(s), nil
}

func (cw *cobraWriter) Println(s ...any) {
	cw.c.Println(s...)
}

// helpWriter supports writing command usage messages.
type helpWriter struct {
	fw    *formattingWriter
	empty bool
}

func makeHelpWriter(w *formattingWriter) *helpWriter {
	return &helpWriter{w, true}
}

// Indent sets indentation for the next WriteLn call.
func (w *helpWriter) Indent(indent ...int) *helpWriter {
	w.fw.SetIndent(indent...)
	return w
}

// WriteLn writes the strings with the underlying writer, adds new line and
// resets the indentation.
func (w *helpWriter) Write(s ...string) error {
	for _, s := range s {
		if len(s) > 0 {
			if err := w.fw.WriteString(s); err != nil {
				return err
			}
			w.empty = false
		}
	}
	return nil
}

// WriteLn writes the strings with the underlying writer, adds new line and
// resets the indentation.
func (w *helpWriter) WriteLn(s ...string) error {
	err := w.Write(s...)
	if err == nil {
		err = w.Indent().fw.WriteString("\n")
		w.empty = false
	}
	return err
}

func (w *helpWriter) EmptyLineSeparator() error {
	if !w.empty {
		w.empty = true
		return w.Indent().fw.WriteString("\n")
	}
	return nil
}
