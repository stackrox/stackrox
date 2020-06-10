package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLogLineQuotaSetting(t *testing.T) {
	cases := []struct {
		input                  string
		maxLines, intervalSecs int64
		expectErr              bool
	}{
		{
			input: "",
		},
		{
			input: "/",
		},
		{
			input:    "5",
			maxLines: 5,
		},
		{
			input:     "0",
			expectErr: true,
		},
		{
			input:     "-5",
			expectErr: true,
		},
		{
			input:     "foo",
			expectErr: true,
		},
		{
			input:    "5/",
			maxLines: 5,
		},
		{
			input:     "0/",
			expectErr: true,
		},
		{
			input:     "-5/",
			expectErr: true,
		},
		{
			input:     "foo/",
			expectErr: true,
		},
		{
			input:        "/15",
			intervalSecs: 15,
		},
		{
			input:        "/ 15",
			intervalSecs: 15,
		},
		{
			input:     "/-15",
			expectErr: true,
		},
		{
			input:     "/ 0",
			expectErr: true,
		},
		{
			input:     "/ bar",
			expectErr: true,
		},
		{
			input:        "5/15",
			maxLines:     5,
			intervalSecs: 15,
		},
		{
			input:        "5 / 15",
			maxLines:     5,
			intervalSecs: 15,
		},
		{
			input:     "0 / 15",
			expectErr: true,
		},
		{
			input:     "-1 / 15",
			expectErr: true,
		},
		{
			input:     "foo / 15",
			expectErr: true,
		},
		{
			input:     "5 / 0",
			expectErr: true,
		},
		{
			input:     "5 / -15",
			expectErr: true,
		},
		{
			input:     "5 / bar",
			expectErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			maxLines, intervalSecs, err := parseLogLineQuotaSetting(c.input)
			assert.Equal(t, c.maxLines, maxLines)
			assert.Equal(t, c.intervalSecs, intervalSecs)
			if c.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
