package utils

// helpWriter supports writing command usage messages.
type helpWriter struct {
	fw    *formattingWriter
	empty bool
	err   error
}

func makeHelpWriter(w *formattingWriter) *helpWriter {
	return &helpWriter{w, true, nil}
}

// Indent sets indentation for the next WriteLn call.
func (w *helpWriter) Indent(indent ...int) *helpWriter {
	w.fw.SetIndent(indent...)
	return w
}

// WriteLn writes the strings with the underlying writer, adds new line and
// resets the indentation.
func (w *helpWriter) Write(s ...string) {
	if w.err != nil {
		return
	}
	for _, s := range s {
		if len(s) > 0 {
			w.err = w.fw.WriteString(s)
			if w.err != nil {
				return
			}
			w.empty = false
		}
	}
}

// WriteLn writes the strings with the underlying writer, adds new line and
// resets the indentation.
func (w *helpWriter) WriteLn(s ...string) {
	w.Write(s...)
	if w.err == nil {
		w.Indent().Write("\n")
		w.empty = false
	}
}

func (w *helpWriter) EmptyLineSeparator() {
	if !w.empty {
		w.WriteLn()
		w.empty = true
	}
}
