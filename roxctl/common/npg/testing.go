package npg

import (
	goerrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertErrorsContain is a test helper that asserts that all expectedStrings are included in gotSubject.
// The prefix is used to name the subject under test when multiple are tested in a row (e.g., errors, warnings).
func AssertErrorsContain(t *testing.T, expectedStrings []string, gotSubject []error, prefix string) {
	require.Lenf(t, gotSubject, len(expectedStrings), "number of %s should be %d", prefix, len(expectedStrings))

	for _, expError := range expectedStrings {
		if expError != "" {
			require.Error(t, goerrors.Join(gotSubject...))
			assert.ErrorContainsf(t, goerrors.Join(gotSubject...), expError,
				"Expected %s to contain %s", prefix, expError)
		} else {
			assert.NoError(t, goerrors.Join(gotSubject...))
		}
	}
}
