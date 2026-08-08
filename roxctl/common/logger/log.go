package logger

import (
	"fmt"

	io2 "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
)

// Logger is a struct responsible for printing messages. It should be preferred over fmt functions.
type Logger interface {
	// ErrfLn prints a formatted string with a newline, prefixed with ERROR and colorized
	ErrfLn(format string, a ...any)

	// WarnfLn prints a formatted string with a newline, prefixed with WARN and colorized
	WarnfLn(format string, a ...any)

	// InfofLn prints a formatted string with a newline, prefixed with INFO and colorized
	InfofLn(format string, a ...any)

	// PrintfLn prints a formatted string with newline at the end
	PrintfLn(format string, a ...any)
}

type logger struct {
	io      io2.IO
	printer printer.ColorfulPrinter
}

// NewLogger returns new instance of Logger
func NewLogger(io io2.IO, colorfulPrinter printer.ColorfulPrinter) Logger {
	return &logger{
		io:      io,
		printer: colorfulPrinter,
	}
}

func (l *logger) ErrfLn(format string, a ...any) {
	l.printer.Err(l.io.ErrOut(), format+"\n", a...)
}

func (l *logger) WarnfLn(format string, a ...any) {
	l.printer.Warn(l.io.ErrOut(), format+"\n", a...)
}

func (l *logger) InfofLn(format string, a ...any) {
	l.printer.Info(l.io.ErrOut(), format+"\n", a...)
}

func (l *logger) PrintfLn(format string, a ...any) {
	_, _ = fmt.Fprint(l.io.Out(), l.colorWords(format+"\n", a...))
}

func (l *logger) colorWords(format string, a ...any) string {
	str := fmt.Sprintf(format, a...)
	return l.printer.ColorWords(str)
}
