package enforcers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBypass(t *testing.T) {
	var cases = []struct {
		annotations map[string]string
		expected    bool
	}{
		{
			annotations: map[string]string{
				"lol": "hey",
				"app": "central",
			},
			expected: true,
		},
		{
			annotations: map[string]string{
				"lol":                               "hey",
				"admission.stackrox.io/break-glass": "break-glass",
			},
			expected: false,
		},
		{
			annotations: map[string]string{
				"lol":                               "hey",
				"admission.stackrox.io/break-glass": "",
			},
			expected: false,
		},
		{
			annotations: map[string]string{
				"lol":                   "hey",
				"admission.stackrox.io": "dont-break-glass",
			},
			expected: true,
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Annotations-%d", i), func(t *testing.T) {
			assert.Equal(t, c.expected, ShouldEnforce(c.annotations))
		})
	}
}
