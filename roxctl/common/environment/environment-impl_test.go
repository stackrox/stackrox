package environment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func Test_determineAuthMethodEx(t *testing.T) {
	if buildinfo.ReleaseBuild {
		t.SkipNow()
	}

	const missing = false
	const changed = true
	const empty = true
	const value = false

	tests := map[string]struct {
		tokenFileChanged   bool
		passwordChanged    bool
		tokenFileNameEmpty bool
		passwordEmpty      bool

		expectedError  error
		expectedMethod string
	}{
		"0":       {missing, missing, empty, empty, nil, ""},
		"1":       {missing, missing, empty, value, nil, "basic auth"},
		"2":       {missing, missing, value, empty, nil, "token based auth"},
		"3":       {missing, missing, value, value, errox.InvalidArgs, ""},
		"4 panic": {missing, changed, empty, empty, nil, ""},
		"5":       {missing, changed, empty, value, nil, "basic auth"},
		"6 panic": {missing, changed, value, empty, nil, ""},
		"7":       {missing, changed, value, value, nil, "basic auth"},
		"8 panic": {changed, missing, empty, empty, nil, ""},
		"9 panic": {changed, missing, empty, value, nil, ""},
		"A":       {changed, missing, value, empty, nil, "token based auth"},
		"B":       {changed, missing, value, value, nil, "token based auth"},
		"C panic": {changed, changed, empty, empty, nil, ""},
		"D panic": {changed, changed, empty, value, nil, ""},
		"E panic": {changed, changed, value, empty, nil, ""},
		"F":       {changed, changed, value, value, errox.InvalidArgs, ""},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if strings.HasSuffix(name, "panic") {
				assert.Panics(t, func() {
					_, _ = determineAuthMethodExt(test.tokenFileChanged, test.passwordChanged, test.tokenFileNameEmpty, test.passwordEmpty)
				})
				return
			}
			method, err := determineAuthMethodExt(test.tokenFileChanged, test.passwordChanged, test.tokenFileNameEmpty, test.passwordEmpty)
			require.ErrorIs(t, err, test.expectedError, err)
			if test.expectedMethod != "" {
				require.NotNil(t, method)
				assert.Equal(t, test.expectedMethod, method.Type())
			} else {
				if !assert.Nil(t, method) {
					t.Log(method.Type())
				}
			}
		})
	}
}
