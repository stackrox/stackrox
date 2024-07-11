package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllDefaultPoliciesValid(t *testing.T) {
	defaultPolicies, err := policies.DefaultPolicies()
	require.NoError(t, err)

	for _, p := range defaultPolicies {
		t.Run(p.GetName(), func(t *testing.T) {
			assert.NoError(t, Validate(p), p)
		})
	}
}
