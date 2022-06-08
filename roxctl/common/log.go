package common

import (
	"fmt"

	"github.com/stackrox/rox/roxctl/common/printer"
)

type logger struct {
	io      IO
	printer printer.ColorfulPrinter
}

// NewLogger returns new instance of Logger
func NewLogger(io IO, colorfulPrinter printer.ColorfulPrinter) Logger {
	return &logger{
		io:      io,
		printer: colorfulPrinter,
	}
}

func (l *logger) ErrfLn(format string, a ...interface{}) {
	l.printer.Err(l.io.ErrOut(), format+"\n", a...)
}

func (l *logger) WarnfLn(format string, a ...interface{}) {
	l.printer.Warn(l.io.ErrOut(), format+"\n", a...)
}

func (l *logger) InfofLn(format string, a ...interface{}) {
	l.printer.Info(l.io.ErrOut(), format+"\n", a...)
}

func (l *logger) PrintfLn(format string, a ...interface{}) {
	_, _ = fmt.Fprint(l.io.Out(), l.colorWords(format+"\n", a...))
}

func (l *logger) colorWords(format string, a ...interface{}) string {
	str := fmt.Sprintf(format, a...)
	return l.printer.ColorWords(str)
}
