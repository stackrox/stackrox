package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPprofAddr(t *testing.T) {
	cases := map[string]struct {
		input    string
		expected string
	}{
		"port only":      {input: ":8443", expected: ":9443"},
		"host and port":  {input: "localhost:8443", expected: "localhost:9443"},
		"IPv6 with port": {input: "[::1]:8443", expected: "[::1]:9443"},
		"bare IPv6":      {input: "::1", expected: "[::1]:9443"},
		"IPv4 with port": {input: "10.0.0.1:8443", expected: "10.0.0.1:9443"},
		"full IPv6":      {input: "[2001:db8::1]:8443", expected: "[2001:db8::1]:9443"},
		"bare full IPv6": {input: "2001:db8::1", expected: "[2001:db8::1]:9443"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, pprofAddr(tc.input))
		})
	}
}
