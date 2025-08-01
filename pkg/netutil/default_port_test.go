package netutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithDefaultPort(t *testing.T) {

	cases := []struct {
		input    string
		expected string
	}{
		{input: "", expected: ""},
		{input: "192.168.0.1", expected: "192.168.0.1:1337"},
		{input: "192.168.0.1:31337", expected: "192.168.0.1:31337"},
		{input: "foo.bar", expected: "foo.bar:1337"},
		{input: "foo.bar:31337", expected: "foo.bar:31337"},
		{input: "::1", expected: "[::1]:1337"},
		{input: "[::1]", expected: "[::1]:1337"},
		{input: "[::1]:31337", expected: "[::1]:31337"},
	}

	for _, c := range cases {
		// Test names must not contain colons.
		t.Run(strings.ReplaceAll(c.input, ":", ";"), func(t *testing.T) {
			assert.Equal(t, c.expected, WithDefaultPort(c.input, 1337))
		})
	}
}
