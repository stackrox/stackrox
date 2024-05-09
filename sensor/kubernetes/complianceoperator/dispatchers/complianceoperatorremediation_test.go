package dispatchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveSuffix(t *testing.T) {
	testCases := map[string]struct {
		input    string
		expected string
	}{
		"should cut the suffix":    {input: "test-1", expected: "test"},
		"is one word":              {input: "oneword", expected: "oneword"},
		"has a dash but no number": {input: "something-with-a-dash", expected: "something-with-a-dash"},
		"with one dash":            {input: "one-dash", expected: "one-dash"},
		"with numbers":             {input: "with-123-number", expected: "with-123-number"},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			r := removeSuffix(test.input)
			assert.Equal(t, test.expected, r)
		})
	}
}
