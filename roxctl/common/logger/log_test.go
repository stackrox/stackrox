//go:build linux || darwin

package logger

import (
	"testing"

	"github.com/fatih/color"
	io2 "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	color.NoColor = false
	t.Cleanup(func() {
		color.NoColor = true
	})

	in := "(TOTAL: 0, LOW: 0, MEDIUM: 0, HIGH: 0, CRITICAL: 0)"

	testCases := []struct {
		name   string
		fun    func(l Logger)
		errOut string
		out    string
	}{
		{
			name: "Print",
			fun:  func(l Logger) { l.PrintfLn(in) },
			out:  "(TOTAL: 0, \u001B[34;2mLOW\u001B[0;22m: 0, \u001B[33mMEDIUM\u001B[0m: 0, \u001B[95mHIGH\u001B[0m: 0, \u001B[31;1mCRITICAL\u001B[0;22m: 0)\n",
		},
		{
			name:   "Info",
			fun:    func(l Logger) { l.InfofLn(in) },
			errOut: "\x1b[94mINFO:\t(TOTAL: 0, LOW: 0, MEDIUM: 0, HIGH: 0, CRITICAL: 0)\n\x1b[0m",
		},
		{
			name:   "Warn",
			fun:    func(l Logger) { l.WarnfLn(in) },
			errOut: "\x1b[95mWARN:\t(TOTAL: 0, LOW: 0, MEDIUM: 0, HIGH: 0, CRITICAL: 0)\n\x1b[0m",
		},
		{
			name:   "Error",
			fun:    func(l Logger) { l.ErrfLn(in) },
			errOut: "\x1b[31;1mERROR:\t(TOTAL: 0, LOW: 0, MEDIUM: 0, HIGH: 0, CRITICAL: 0)\n\x1b[0m",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.out+tc.errOut, func(t *testing.T) {
			io, _, out, errOut := io2.TestIO()
			logger := NewLogger(io, printer.DefaultColorPrinter())
			tc.fun(logger)
			assert.Equal(t, tc.out, out.String())
			assert.Equal(t, tc.errOut, errOut.String())
		})
	}
}
