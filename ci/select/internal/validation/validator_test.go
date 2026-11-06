package validation

// validator_test.go pins the seam contract itself: implementations
// must be interchangeable behind Validate, and Trivial must hold its
// end of the fail-closed bargain — an empty report, never a wrong one.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrivialAcceptsEveryDocument(t *testing.T) {
	t.Parallel()
	for name, data := range map[string]string{
		"should accept documents full validation rejects":   "version: 2\njobs: []\nrules: []\n",
		"should accept non-yaml documents":                  "[",
		"should accept empty documents":                     "",
		"should accept documents the engine cannot compile": "version: 1\njobs: [go]\nrules:\n  - name: bad\n    when: any-file-matches\n    paths: ['^[broken'\n    run: [go]\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			warnings, err := Trivial{}.Validate([]byte(data))
			require.NoError(t, err)
			require.Empty(t, warnings)
		})
	}
}

// TestValidatorsShareOneContract runs every implementation over the
// same documents through the interface, so a new implementation slots
// in without caller changes.
func TestValidatorsShareOneContract(t *testing.T) {
	t.Parallel()
	version2 := "version: 2\njobs: [go, postgres]\nrules: []\n"
	for name, c := range map[string]struct {
		validator Validator
		wantError bool
	}{
		"full should keep rejecting unknown versions":  {Full{}, true},
		"trivial should keep accepting every document": {Trivial{}, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := c.validator.Validate([]byte(version2))
			if c.wantError {
				require.ErrorContains(t, err, "unsupported rules version 2")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
