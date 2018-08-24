package blevesearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNumericPrefix(t *testing.T) {
	var cases = []struct {
		value          string
		expectedPrefix string
		expectedValue  string
	}{
		{
			value:          "<=lol",
			expectedPrefix: "<=",
			expectedValue:  "lol",
		},
		{
			value:          ">lol",
			expectedPrefix: ">",
			expectedValue:  "lol",
		},
		{
			value:          ">=lol",
			expectedPrefix: ">=",
			expectedValue:  "lol",
		},
		{
			value:          ">lol",
			expectedPrefix: ">",
			expectedValue:  "lol",
		},
	}
	for _, c := range cases {
		t.Run(c.value, func(t *testing.T) {
			prefix, value := parseNumericPrefix(c.value)
			assert.Equal(t, c.expectedPrefix, prefix)
			assert.Equal(t, c.expectedValue, value)
		})
	}
}
