package walker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeSingular(t *testing.T) {
	testCases := []struct {
		name           string
		sample         string
		expectedOutput string
	}{
		{
			name:           "Empty string stays empty",
			sample:         "",
			expectedOutput: "",
		},
		{
			name:           "single-character string (random not \"s\") stays the same",
			sample:         "a",
			expectedOutput: "a",
		},
		{
			name:           "single-character string (random not \"s\") stays the same",
			sample:         "Z",
			expectedOutput: "Z",
		},
		{
			name:           "single-character string (\"s\") stays the same",
			sample:         "s",
			expectedOutput: "s",
		},
		{
			name:           "single-character string (\"S\") stays the same",
			sample:         "S",
			expectedOutput: "S",
		},
		{
			name:           "singular word (not ending with \"s\") stays the same",
			sample:         "example",
			expectedOutput: "example",
		},
		{
			name:           "plural word (ending with \"s\") gets corrected",
			sample:         "deployments",
			expectedOutput: "deployment",
		},
	}

	for _, c := range testCases {
		assert.Equal(t, c.expectedOutput, makeSingular(c.sample), c.name)
	}
}
