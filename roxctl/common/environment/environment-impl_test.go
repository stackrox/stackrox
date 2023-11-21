package environment

import (
	"fmt"
	"testing"

	"github.com/fatih/color"
	"github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
)

func Test_colorWriter_Write(t *testing.T) {
	color.NoColor = false
	t.Cleanup(func() {
		color.NoColor = true
	})
	tests := []struct {
		given    string
		expected string
	}{
		{
			given:    "(TOTAL: 0, LOW: 0, MEDIUM: 0, HIGH: 0, CRITICAL: 0)",
			expected: "(TOTAL: 0, \x1b[34;2mLOW\x1b[0;22m: 0, \x1b[33mMEDIUM\x1b[0m: 0, \x1b[95mHIGH\x1b[0m: 0, \x1b[31;1mCRITICAL\x1b[0;22m: 0)",
		},
		{
			given:    "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
			expected: "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
		},
		{
			given:    "HIGHSCORE",
			expected: "\u001B[95mHIGH\u001B[0mSCORE",
		},
	}
	for _, tt := range tests {
		c := tt
		t.Run(c.given, func(t *testing.T) {
			testIO, _, testStdOut, _ := io.TestIO()
			env := NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

			w := env.ColorWriter()
			n, err := fmt.Fprint(w, c.given)

			assert.NoError(t, err)
			assert.Len(t, c.given, n)
			assert.Equal(t, c.expected, testStdOut.String())
		})
	}
}
