package awssh

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllDefaultCategoriesHaveMappings(t *testing.T) {
	defaults.PoliciesPath = policies.Directory()

	defaultPolicies, err := defaults.Policies()
	require.NoError(t, err)

	categoryMapSet := set.NewStringSet()
	for k := range categoryMap {
		categoryMapSet.Add(k)
	}

	for _, policy := range defaultPolicies {
		for _, category := range policy.GetCategories() {
			_, ok := categoryMap[strings.ToLower(category)]
			if ok {
				categoryMapSet.Remove(strings.ToLower(category))
			}
			assert.True(t, ok, "category %s not mapped", category)
		}
	}
	// Ensure that all categories in the map are used in policies
	assert.Len(t, categoryMapSet, 0)
}
