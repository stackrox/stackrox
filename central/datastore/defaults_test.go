package datastore

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/central/detection/matcher"
	"bitbucket.org/stack-rox/apollo/image/policies"
	"bitbucket.org/stack-rox/apollo/pkg/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultPolicies(t *testing.T) {
	defaults.PoliciesPath = policies.Directory()
	policies, err := defaults.Policies()
	require.NoError(t, err)
	require.NotNil(t, policies)

	for _, p := range policies {
		_, err := matcher.New(p)
		assert.NoError(t, err)
	}
}
