package effectiveaccessscope

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func TestSelectorsMatchCluster(t *testing.T) {
	focusOnMelangeRequirement, err := labels.NewRequirement("focus", selection.Equals, []string{"melange"})
	require.NoError(t, err)
	require.NotNil(t, focusOnMelangeRequirement)

	for name, tc := range map[string]struct {
		ruleSelector *selectors
		cluster      *storage.Cluster
		expected     scopeState
	}{
		"nil selector always excludes cluster": {
			ruleSelector: nil,
			cluster:      clusterEarth,
			expected:     Excluded,
		},
		"cluster matched by ID is included": {
			ruleSelector: &selectors{
				clustersByID: set.NewStringSet(clusterEarth.GetId()),
			},
			cluster:  clusterEarth,
			expected: Included,
		},
		"cluster matched by name (matching k8s syntax) is included": {
			ruleSelector: &selectors{
				clustersByName: set.NewStringSet(clusterEarth.GetName()),
			},
			cluster:  clusterEarth,
			expected: Included,
		},
		"cluster matched by name (NOT matching k8s syntax) is included": {
			ruleSelector: &selectors{
				clustersByName: set.NewStringSet(clusterGiediPrime.GetName()),
			},
			cluster:  clusterGiediPrime,
			expected: Included,
		},
		"cluster matched by label is included": {
			ruleSelector: &selectors{
				clustersByLabel: []labels.Selector{
					labels.NewSelector().Add(*focusOnMelangeRequirement),
				},
			},
			cluster:  clusterArrakis,
			expected: Included,
		},
		"cluster NOT matched by label is excluded": {
			ruleSelector: &selectors{
				clustersByLabel: []labels.Selector{
					labels.NewSelector().Add(*focusOnMelangeRequirement),
				},
			},
			cluster:  clusterEarth,
			expected: Excluded,
		},
	} {
		t.Run(name, func(it *testing.T) {
			result := tc.ruleSelector.matchCluster(tc.cluster)
			assert.Equal(it, tc.expected, result)
		})
	}
}

func TestSelectorsMatchNamespace(t *testing.T) {
	focusOnMelangeRequirement, err := labels.NewRequirement("focus", selection.Equals, []string{"melange"})
	require.NoError(t, err)
	require.NotNil(t, focusOnMelangeRequirement)

	for name, tc := range map[string]struct {
		ruleSelectors *selectors
		namespace     *storage.NamespaceMetadata
		expected      scopeState
	}{
		"nil selector always exclude namespaces": {
			ruleSelectors: nil,
			namespace:     nsSkunkWorks,
			expected:      Excluded,
		},
		"namespace matched by cluster ID is included": {
			ruleSelectors: &selectors{
				namespacesByClusterID: map[string]set.StringSet{
					nsSkunkWorks.GetClusterId(): set.NewStringSet(nsSkunkWorks.GetName()),
				},
			},
			namespace: nsSkunkWorks,
			expected:  Included,
		},
		"namespace matched by cluster name is included": {
			ruleSelectors: &selectors{
				namespacesByClusterName: map[string]set.StringSet{
					nsSkunkWorks.GetClusterName(): set.NewStringSet(nsSkunkWorks.GetName()),
				},
			},
			namespace: nsSkunkWorks,
			expected:  Included,
		},
		"namespace matched by label is included": {
			ruleSelectors: &selectors{
				namespacesByLabel: []labels.Selector{
					labels.NewSelector().Add(*focusOnMelangeRequirement),
				},
			},
			namespace: nsAtreides,
			expected:  Included,
		},
		"namespace NOT matched by label is excluded": {
			ruleSelectors: &selectors{
				namespacesByLabel: []labels.Selector{
					labels.NewSelector().Add(*focusOnMelangeRequirement),
				},
			},
			namespace: nsSkunkWorks,
			expected:  Excluded,
		},
	} {
		t.Run(name, func(it *testing.T) {
			result := tc.ruleSelectors.matchNamespace(tc.namespace)
			assert.Equal(it, tc.expected, result)
		})
	}
}
