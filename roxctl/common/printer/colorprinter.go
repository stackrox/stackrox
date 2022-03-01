package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

const (
	errorMsgPrefix = "ERROR:\t"
	warnMsgPrefix  = "WARN:\t"
	infoMsgPrefix  = "INFO:\t"
)

var (
	colorKeyWordMap = map[string][]color.Attribute{
		"LOW":       {color.FgBlue, color.Faint},
		"MEDIUM":    {color.FgYellow},
		"HIGH":      {color.FgHiMagenta},
		"CRITICAL":  {color.FgRed, color.Bold},
		"IMPORTANT": {color.FgRed},
	}
)

// ColorfulPrinter abstracts colorful printing for error, warning and info messages. All messages are prefixed
// with either "ERROR:", "WARN:" or "INFO:"
type ColorfulPrinter interface {
	// Err prints a formatted string to the io.Writer, prefixed with ERROR and colorized
	Err(out io.Writer, format string, a ...interface{})

	// Warn prints a formatted string to the io.Writer, prefixed with WARN and colorized
	Warn(out io.Writer, format string, a ...interface{})

	// Info prints a formatted string to the io.Writer, prefixed with INFO and colorized
	Info(out io.Writer, format string, a ...interface{})

	// ColorWords colors a words with a specific color and returns the colorized string
	ColorWords(s string) string
}

type colorPrinter struct {
	err             func(w io.Writer, format string, a ...interface{})
	warn            func(w io.Writer, format string, a ...interface{})
	info            func(w io.Writer, format string, a ...interface{})
	bold            func(w io.Writer, format string, a ...interface{})
	colorKeyWordMap map[string]string
}

// DefaultColorPrinter creates a ColorfulPrinter with default colors to print messages.
func DefaultColorPrinter() ColorfulPrinter {
	wordToColorfulWord := make(map[string]string, len(colorKeyWordMap))
	for w, attr := range colorKeyWordMap {
		wordToColorfulWord[w] = color.New(attr...).Sprintf(w)
	}
	c := &colorPrinter{
		err:             color.New(color.FgRed, color.Bold).FprintfFunc(),
		warn:            color.New(color.FgHiMagenta).FprintfFunc(),
		info:            color.New(color.FgHiBlue).FprintfFunc(),
		bold:            color.New(color.Bold).FprintfFunc(),
		colorKeyWordMap: wordToColorfulWord,
	}
	return c
}

// NoColorPrinter is a printer that does not support colors.
func NoColorPrinter() ColorfulPrinter {
	printf := func(w io.Writer, format string, a ...interface{}) {
		_, _ = fmt.Fprintf(w, format, a...)
	}
	c := &colorPrinter{
		err:  printf,
		warn: printf,
		info: printf,
		bold: printf,
	}
	return c
}

func (c *colorPrinter) Err(out io.Writer, format string, a ...interface{}) {
	c.err(out, errorMsgPrefix+format, a...)
}

func (c *colorPrinter) Warn(out io.Writer, format string, a ...interface{}) {
	c.warn(out, warnMsgPrefix+format, a...)
}

func (c *colorPrinter) Info(out io.Writer, format string, a ...interface{}) {
	c.info(out, infoMsgPrefix+format, a...)
}

func (c *colorPrinter) ColorWords(s string) string {
	for w, colorWord := range c.colorKeyWordMap {
		s = strings.ReplaceAll(s, w, colorWord)
	}
	return s
}
