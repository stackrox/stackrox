package rulesvalidation

// contract_test.go checks the concrete validators against the contract
// the engine hands out: YAML and TrivialValidator must be interchangeable
// behind engine.Validator, so a language can offer either depth.

import (
	"testing"

	"github.com/stackrox/rox/ci/select/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestValidatorsShareOneContract(t *testing.T) {
	t.Parallel()
	version2 := "version: 2\njobs: [go, postgres]\nrules: []\n"
	for name, c := range map[string]struct {
		validator engine.Validator
		wantError bool
	}{
		"yaml should keep accepting parsable documents": {YAML{}, false},
		"trivial should keep accepting every document":  {engine.TrivialValidator{}, false},
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
