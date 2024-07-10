//go:build test_all

package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFriendlyParseBool(t *testing.T) {
	cases := []struct {
		input  string
		output bool
		err    bool
	}{
		{
			input:  "true",
			output: true,
			err:    false,
		},
		{
			input:  "tr",
			output: true,
			err:    false,
		},
		{
			input:  "t",
			output: true,
			err:    false,
		},
		{
			input:  "false",
			output: false,
			err:    false,
		},
		{
			input:  "fa",
			output: false,
			err:    false,
		},
		{
			input:  "f",
			output: false,
			err:    false,
		},
		{
			input:  "",
			output: false,
			err:    true,
		},
		{
			input:  "lol",
			output: false,
			err:    true,
		},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			output, err := FriendlyParseBool(c.input)
			assert.Equal(t, c.output, output)
			assert.Equal(t, c.err, err != nil)
		})
	}
}
