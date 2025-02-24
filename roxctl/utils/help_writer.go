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

func makeHelpWriter(w io.StringWriter, width int, indent ...int) *helpWriter {
	return &helpWriter{makeWriter(w, width, indent...), true}
}

// Indent sets indentation for the next WriteLn call.
func (w *helpWriter) Indent(indent ...int) *helpWriter {
	w.fw.setIndent(indent...)
	return w
}

// WriteLn writes the strings with the underlying writer, adds new line and
// resets the indentation.
func (w *helpWriter) Write(s ...string) {
	for _, s := range s {
		if len(s) > 0 {
			_, _ = w.fw.WriteString(s)
		}
		w.empty = false
	}
}

// WriteLn writes the strings with the underlying writer, adds new line and
// resets the indentation.
func (w *helpWriter) WriteLn(s ...string) {
	w.Write(s...)
	w.fw.setIndent()
	_, _ = w.fw.WriteString("\n")
	w.empty = false
}

func (w *helpWriter) EmptyLineSeparator() {
	if !w.empty {
		w.fw.setIndent()
		_, _ = w.fw.WriteString("\n")
		w.empty = true
	}
}
