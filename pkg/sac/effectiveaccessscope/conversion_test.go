package effectiveaccessscope

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
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
		"empty ruleset result in an empty selector": {
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
		// cluster selection by ID
		"empty included cluster ID rules leave the clustersByID part of the selector empty": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedClusterIds: make([]string, 0),
			},
			expected: emptySelector(),
		},
		"included cluster ID rules fill in the clustersByID part of the selector": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedClusterIds: []string{
					fixtureconsts.Cluster1,
					fixtureconsts.Cluster2,
				},
			},
			expected: selectOnlyClustersByID([]string{fixtureconsts.Cluster1, fixtureconsts.Cluster2}),
		},
		"included cluster ID rules get deduplicated in the clustersByID part of the selector": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedClusterIds: []string{
					fixtureconsts.Cluster1,
					fixtureconsts.Cluster1,
				},
			},
			expected: selectOnlyClustersByID([]string{fixtureconsts.Cluster1}),
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
				nil,
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
				nil,
				map[string][]string{
					clusterName1: {namespaceName1},
				},
			),
		},
		// namespace selection by cluster id and namespace name
		"namespace selection rules by cluster ID fill in the selector namespacesByClusterID": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{
						ClusterId:     fixtureconsts.Cluster1,
						NamespaceName: namespaceName1,
					},
					{
						ClusterId:     fixtureconsts.Cluster2,
						NamespaceName: namespaceName2,
					},
				},
			},
			expected: selectNamespacesByCluster(
				map[string][]string{
					fixtureconsts.Cluster1: {namespaceName1},
					fixtureconsts.Cluster2: {namespaceName2},
				},
				nil,
			),
		},
		"namespace selection rules by cluster ID get deduplicated in the selector namespacesByClusterID": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{
						ClusterId:     fixtureconsts.Cluster1,
						NamespaceName: namespaceName1,
					},
					{
						ClusterId:     fixtureconsts.Cluster1,
						NamespaceName: namespaceName1,
					},
				},
			},
			expected: selectNamespacesByCluster(
				map[string][]string{fixtureconsts.Cluster1: {namespaceName1}},
				nil,
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
						ClusterId: fixtureconsts.Cluster1,
					},
					{
						ClusterName: clusterName2,
					},
				},
			},
			expected: emptySelector(),
		},
		// namespace explicit selection favors cluster ID over cluster name when both are available
		"namespace selection rules by cluster ID and name fill in the selector namespacesByClusterID": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					{
						ClusterId:     fixtureconsts.Cluster1,
						ClusterName:   clusterName1,
						NamespaceName: namespaceName1,
					},
					{
						ClusterId:     fixtureconsts.Cluster2,
						ClusterName:   clusterName2,
						NamespaceName: namespaceName2,
					},
				},
			},
			expected: selectNamespacesByCluster(
				map[string][]string{
					fixtureconsts.Cluster1: {namespaceName1},
					fixtureconsts.Cluster2: {namespaceName2},
				},
				nil,
			),
		},
		// mix of multiple rules
		"mix of selection rules": {
			rules: &storage.SimpleAccessScope_Rules{
				IncludedClusterIds: []string{fixtureconsts.Cluster1, fixtureconsts.Cluster2, fixtureconsts.Cluster1},
				IncludedClusters:   []string{clusterName2, clusterName1, clusterName2},
				IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
					nil,
					{},
					{
						ClusterName:   clusterName1,
						NamespaceName: namespaceName1,
					},
					{
						ClusterId:     fixtureconsts.Cluster2,
						NamespaceName: namespaceName1,
					},
					{
						ClusterId:     fixtureconsts.Cluster1,
						ClusterName:   clusterName1,
						NamespaceName: namespaceName2,
					},
				},
				ClusterLabelSelectors:   nil,
				NamespaceLabelSelectors: nil,
			},
			expected: &selectors{
				clustersByID: map[string]bool{
					fixtureconsts.Cluster1: true,
					fixtureconsts.Cluster2: true,
				},
				clustersByName: map[string]bool{
					clusterName1: true,
					clusterName2: true,
				},
				clustersByLabel: make([]labels.Selector, 0),
				namespacesByClusterID: map[string]map[string]bool{
					fixtureconsts.Cluster1: {namespaceName2: true},
					fixtureconsts.Cluster2: {namespaceName1: true},
				},
				namespacesByClusterName: map[string]map[string]bool{
					clusterName1: {namespaceName1: true},
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
		clustersByID:            make(map[string]bool),
		clustersByName:          make(map[string]bool),
		clustersByLabel:         make([]labels.Selector, 0),
		namespacesByClusterID:   make(map[string]map[string]bool),
		namespacesByClusterName: make(map[string]map[string]bool),
		namespacesByLabel:       make([]labels.Selector, 0),
	}
}

func selectOnlyClustersByID(clusterIDs []string) *selectors {
	selector := emptySelector()
	for _, clusterID := range clusterIDs {
		selector.clustersByID[clusterID] = true
	}
	return selector
}

func selectOnlyClustersByName(clusterNames []string) *selectors {
	selector := emptySelector()
	for _, clusterName := range clusterNames {
		selector.clustersByName[clusterName] = true
	}
	return selector
}

func selectNamespacesByCluster(
	namespacesByClusterID map[string][]string,
	namespacesByClusterName map[string][]string,
) *selectors {
	selector := emptySelector()
	for clusterID, clusterNamespaces := range namespacesByClusterID {
		selector.namespacesByClusterID[clusterID] = make(map[string]bool)
		for _, ns := range clusterNamespaces {
			selector.namespacesByClusterID[clusterID][ns] = true
		}
	}
	for clusterName, clusterNamespaces := range namespacesByClusterName {
		selector.namespacesByClusterName[clusterName] = make(map[string]bool)
		for _, ns := range clusterNamespaces {
			selector.namespacesByClusterName[clusterName][ns] = true
		}
	}
	return selector
}
