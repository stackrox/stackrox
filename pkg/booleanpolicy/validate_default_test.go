// This uses a separate package to avoid import cycles with pkg/defaults.
package booleanpolicy_test

import (
	"testing"

	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllDefaultPoliciesValid(t *testing.T) {
	defaults.PoliciesPath = policies.Directory()
	defaultPolicies, err := defaults.Policies()
	require.NoError(t, err)

	for _, p := range defaultPolicies {
		t.Run(p.GetName(), func(t *testing.T) {
			assert.NoError(t, booleanpolicy.Validate(p), p)
		})
	}
}
