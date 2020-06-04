// This uses a separate package to avoid import cycles with pkg/defaults.
package booleanpolicy_test

import (
	"testing"

	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllDefaultPoliciesValid(t *testing.T) {
	envIsolator := testutils.NewEnvIsolator(t)
	defer envIsolator.RestoreAll()
	envIsolator.Setenv(features.BooleanPolicyLogic.EnvVar(), "true")

	defaults.PoliciesPath = policies.Directory()
	defaultPolicies, err := defaults.Policies()
	require.NoError(t, err)

	for _, p := range defaultPolicies {
		t.Run(p.GetName(), func(t *testing.T) {
			assert.NoError(t, booleanpolicy.Validate(p), p)
		})
	}
}
