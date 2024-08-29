//go:build test_e2e

package tests

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/stackrox/rox/pkg/testutils/e2etests"
	"github.com/stretchr/testify/assert"
)

var text = []byte(`Here is some text.
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore
magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.
Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`)

func TestLogMatcher(t *testing.T) {
	tests := map[string]struct {
		funcs          []e2etests.LogMatcher
		expectedResult assert.BoolAssertionFunc
		expectedError  assert.ErrorAssertionFunc
		expectedString string
	}{
		"one match": {
			funcs: []e2etests.LogMatcher{
				e2etests.ContainsLineMatching(regexp.MustCompile("sunt in culpa qui officia deserunt")),
			},
			expectedResult: assert.True,
			expectedError:  assert.NoError,
			expectedString: `[contains line matching "sunt in culpa qui officia deserunt"]`,
		},
		"two matches": {
			funcs: []e2etests.LogMatcher{
				e2etests.ContainsLineMatching(regexp.MustCompile("Lorem ipsum dolor")),
				e2etests.ContainsLineMatching(regexp.MustCompile("Duis aute irure")),
			},
			expectedResult: assert.True,
			expectedError:  assert.NoError,
			expectedString: `[contains line matching "Lorem ipsum dolor" contains line matching "Duis aute irure"]`,
		},
		"text divided with newline": {
			funcs: []e2etests.LogMatcher{
				e2etests.ContainsLineMatching(regexp.MustCompile("labore et dolore.*magna aliqua")),
			},
			expectedResult: assert.False,
			expectedError:  assert.NoError,
			expectedString: `[contains line matching "labore et dolore.*magna aliqua"]`,
		},
	}
	r := bytes.NewReader(text)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual, actualErr := e2etests.AllMatch(r, test.funcs...)
			test.expectedResult(t, actual)
			test.expectedError(t, actualErr)
			assert.Equal(t, test.expectedString, fmt.Sprintf("%s", test.funcs))
		})
	}
}
