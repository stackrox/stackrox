package effectiveaccessscope

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	clusterName1 = "cluster-1"
	clusterName2 = "cluster=2"

	namespaceName1 = "namespace-A"
	namespaceName2 = "namespace-B"
)

func TestConvertRulesToSelectors(t *testing.T) {
	// Error cases

	// Success cases
	for name, tc := range map[string]struct {
		rules    *storage.SimpleAccessScope_Rules
		expected *selectors
	}{
		"nil rules result in an empty selector": {
			rules:    nil,
			expected: emptySelector(),
		},
		"empty ruleset results in an empty selector": {
			rules:    &storage.SimpleAccessScope_Rules{},
			expected: emptySelector(),
		},
		// cluster selection by labels
		// cluster selection by name
		"empty included cluster name rules leave the clustersByName part of the selector empty": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: make([]string, 0),
			},
			expected: emptySelector(),
		},
		"included cluster name rules fill in the clustersByName part of the selector": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{
					clusterName1,
					clusterName2,
				},
			},
			expected: selectOnlyClustersByName([]string{clusterName1, clusterName2}),
		},
		"included cluster name rules get deduplicated in the clustersByName part of the selector": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{
					clusterName1,
					clusterName1,
				},
			},
			expected: selectOnlyClustersByName([]string{clusterName1}),
		},
		// namespace selection by labels
		// namespace selection by cluster name and namespace name
		"empty namespace selection rules leave the namespaces parts of the selector empty": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: make([]*storage.SimpleAccessScope_Rules_Namespace, 0),
			},
			expected: emptySelector(),
		},
		"namespace selection rules by cluster name fill in the selector namespacesByClusterName": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{
						ClusterName:   clusterName1,
						NamespaceName: namespaceName1,
					},
					{
						ClusterName:   clusterName2,
						NamespaceName: namespaceName2,
					},
				},
			},
			expected: selectNamespacesByCluster(
				map[string][]string{
					clusterName1: {namespaceName1},
					clusterName2: {namespaceName2},
				},
			),
		},
		"namespace selection rules by cluster name get deduplicated in the selector namespacesByClusterName": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{
						ClusterName:   clusterName1,
						NamespaceName: namespaceName1,
					},
					{
						ClusterName:   clusterName1,
						NamespaceName: namespaceName1,
					},
				},
			},
			expected: selectNamespacesByCluster(
				map[string][]string{
					clusterName1: {namespaceName1},
				},
			),
		},
		// namespace explicit selection with missing cluster or namespace identification are ignored
		"namespace selection rules missing cluster or namespace info are ignored": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{
						NamespaceName: namespaceName1,
					},
					{
						ClusterName: clusterName2,
					},
				},
			},
			expected: emptySelector(),
		},
		// mix of multiple rules
		"mix of selection rules": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{clusterName2, clusterName1, clusterName2},
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					nil,
					{},
					{
						ClusterName:   clusterName1,
						NamespaceName: namespaceName1,
					},
				},
				ClusterLabelSelectors:   nil,
				NamespaceLabelSelectors: nil,
			},
			expected: &selectors{
				clustersByName:  set.NewStringSet(clusterName1, clusterName2),
				clustersByLabel: make([]labels.Selector, 0),
				namespacesByClusterName: map[string]set.StringSet{
					clusterName1: set.NewStringSet(namespaceName1),
				},
				namespacesByLabel: make([]labels.Selector, 0),
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			output, err := convertRulesToSelectors(tc.rules)
			assert.NoError(it, err)
			assert.Equal(it, tc.expected, output)
		})
	}
}

func emptySelector() *selectors {
	return &selectors{
		clustersByName:          set.NewStringSet(),
		clustersByLabel:         make([]labels.Selector, 0),
		namespacesByClusterName: make(map[string]set.StringSet),
		namespacesByLabel:       make([]labels.Selector, 0),
	}
}

func selectOnlyClustersByName(clusterNames []string) *selectors {
	selector := emptySelector()
	selector.clustersByName.AddAll(clusterNames...)
	return selector
}

func selectNamespacesByCluster(
	namespacesByClusterName map[string][]string,
) *selectors {
	selector := emptySelector()
	for clusterName, clusterNamespaces := range namespacesByClusterName {
		selector.namespacesByClusterName[clusterName] = set.NewStringSet(clusterNamespaces...)
	}
	return selector
}
