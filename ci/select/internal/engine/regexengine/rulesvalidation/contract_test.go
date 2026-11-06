package rulesvalidation

// contract_test.go checks the concrete validator against the
// engine-agnostic seam: Full and the neutral Trivial must be
// interchangeable behind validation.Validator, so wiring can swap them
// without touching callers.

import (
	"testing"

	"github.com/stackrox/rox/ci/select/internal/validation"
	"github.com/stretchr/testify/require"
)

func TestValidatorsShareOneContract(t *testing.T) {
	t.Parallel()
	version2 := "version: 2\njobs: [go, postgres]\nrules: []\n"
	for name, c := range map[string]struct {
		validator validation.Validator
		wantError bool
	}{
		"full should keep rejecting unknown versions":  {Full{}, true},
		"trivial should keep accepting every document": {validation.Trivial{}, false},
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
