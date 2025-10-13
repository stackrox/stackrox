//go:build linux || darwin

package printer

import (
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestColorPrinter(t *testing.T) {
	color.NoColor = false
	t.Cleanup(func() {
		color.NoColor = true
	})
	printer := DefaultColorPrinter()
	in := "(TOTAL: 0, LOW: 0, MEDIUM: 0, HIGH: 0, CRITICAL: 0)"
	out := "(TOTAL: 0, \x1b[34;2mLOW\x1b[0;22m: 0, \x1b[33mMEDIUM\x1b[0m: 0, \x1b[95mHIGH\x1b[0m: 0, \x1b[31;1mCRITICAL\x1b[0;22m: 0)"
	assert.Equal(t, out, printer.ColorWords(in))
}

func TestNoColorPrinter(t *testing.T) {
	printer := NoColorPrinter()
	in := "(TOTAL: 0, LOW: 0, MEDIUM: 0, HIGH: 0, CRITICAL: 0)"
	assert.Equal(t, in, printer.ColorWords(in))
}
