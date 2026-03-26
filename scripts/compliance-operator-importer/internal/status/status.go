// Package status provides compact, stage-by-stage progress output
// inspired by modern CLI tools and LLM chat interfaces.
package status

import (
	"fmt"
	"io"
	"os"
)

// Printer writes structured status messages to an output writer.
type Printer struct {
	out io.Writer
}

// New creates a Printer that writes to os.Stderr.
func New() *Printer {
	return &Printer{out: os.Stderr}
}

// NewWithWriter creates a Printer that writes to w.
func NewWithWriter(w io.Writer) *Printer {
	return &Printer{out: w}
}

// Stage prints a stage header: "▸ Stage: message".
func (p *Printer) Stage(stage, msg string) {
	fmt.Fprintf(p.out, "▸ %s: %s\n", stage, msg)
}

// Stagef prints a formatted stage header.
func (p *Printer) Stagef(stage, format string, args ...any) {
	p.Stage(stage, fmt.Sprintf(format, args...))
}

// Detail prints an indented detail line under the current stage.
func (p *Printer) Detail(msg string) {
	fmt.Fprintf(p.out, "  %s\n", msg)
}

// Detailf prints a formatted detail line.
func (p *Printer) Detailf(format string, args ...any) {
	p.Detail(fmt.Sprintf(format, args...))
}

// OK prints a success result for the current stage.
func (p *Printer) OK(msg string) {
	fmt.Fprintf(p.out, "  ✓ %s\n", msg)
}

// OKf prints a formatted success result.
func (p *Printer) OKf(format string, args ...any) {
	p.OK(fmt.Sprintf(format, args...))
}

// Warn prints a warning result for the current stage.
func (p *Printer) Warn(msg string) {
	fmt.Fprintf(p.out, "  ! %s\n", msg)
}

// Warnf prints a formatted warning.
func (p *Printer) Warnf(format string, args ...any) {
	p.Warn(fmt.Sprintf(format, args...))
}

// Fail prints a failure result for the current stage.
func (p *Printer) Fail(msg string) {
	fmt.Fprintf(p.out, "  ✗ %s\n", msg)
}

// Failf prints a formatted failure.
func (p *Printer) Failf(format string, args ...any) {
	p.Fail(fmt.Sprintf(format, args...))
}
