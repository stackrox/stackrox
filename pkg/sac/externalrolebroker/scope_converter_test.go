package externalrolebroker

import (
	"testing"

	"github.com/stackrox/rox/pkg/set"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertBindingsToSimpleAccessScope(t *testing.T) {
	tests := map[string]struct {
		bindings               []clusterviewv1alpha1.ClusterBinding
		expectedClusters       []string
		expectedNamespaceCount int
		checkNamespaces        func(t *testing.T, namespaces []string)
	}{
		"empty bindings": {
			bindings:               []clusterviewv1alpha1.ClusterBinding{},
			expectedClusters:       []string{},
			expectedNamespaceCount: 0,
		},
		"single cluster-scoped binding": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
			},
			expectedClusters:       []string{"cluster-a"},
			expectedNamespaceCount: 0,
		},
		"multiple cluster-scoped bindings": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
				{
					Cluster:    "cluster-b",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
				{
					Cluster:    "cluster-c",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
			},
			expectedClusters:       []string{"cluster-a", "cluster-b", "cluster-c"},
			expectedNamespaceCount: 0,
		},
		"duplicate cluster-scoped bindings": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
			},
			expectedClusters:       []string{"cluster-a"},
			expectedNamespaceCount: 0,
		},
		"single namespace-scoped binding": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"default"},
				},
			},
			expectedClusters:       []string{},
			expectedNamespaceCount: 1,
			checkNamespaces: func(t *testing.T, namespaces []string) {
				assert.Contains(t, namespaces, "cluster-a:default")
			},
		},
		"namespace-scoped binding with multiple namespaces": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"default", "kube-system", "app-namespace"},
				},
			},
			expectedClusters:       []string{},
			expectedNamespaceCount: 3,
			checkNamespaces: func(t *testing.T, namespaces []string) {
				assert.Contains(t, namespaces, "cluster-a:default")
				assert.Contains(t, namespaces, "cluster-a:kube-system")
				assert.Contains(t, namespaces, "cluster-a:app-namespace")
			},
		},
		"multiple namespace-scoped bindings on different clusters": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"default", "app-ns"},
				},
				{
					Cluster:    "cluster-b",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"production", "staging"},
				},
			},
			expectedClusters:       []string{},
			expectedNamespaceCount: 4,
			checkNamespaces: func(t *testing.T, namespaces []string) {
				assert.Contains(t, namespaces, "cluster-a:default")
				assert.Contains(t, namespaces, "cluster-a:app-ns")
				assert.Contains(t, namespaces, "cluster-b:production")
				assert.Contains(t, namespaces, "cluster-b:staging")
			},
		},
		"mixed cluster and namespace scoped bindings": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
				{
					Cluster:    "cluster-b",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"default", "kube-system"},
				},
				{
					Cluster:    "cluster-c",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
			},
			expectedClusters:       []string{"cluster-a", "cluster-c"},
			expectedNamespaceCount: 2,
			checkNamespaces: func(t *testing.T, namespaces []string) {
				assert.Contains(t, namespaces, "cluster-b:default")
				assert.Contains(t, namespaces, "cluster-b:kube-system")
			},
		},
		"namespace-scoped binding with wildcard namespace (should be skipped)": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"*"},
				},
			},
			expectedClusters:       []string{},
			expectedNamespaceCount: 0,
		},
		"namespace-scoped binding with mix of regular and wildcard namespaces": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "cluster-a",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"default", "*", "kube-system"},
				},
			},
			expectedClusters:       []string{},
			expectedNamespaceCount: 2, // Only default and kube-system, wildcard is skipped
			checkNamespaces: func(t *testing.T, namespaces []string) {
				assert.Contains(t, namespaces, "cluster-a:default")
				assert.Contains(t, namespaces, "cluster-a:kube-system")
				assert.NotContains(t, namespaces, "cluster-a:*")
			},
		},
		"complex scenario with multiple bindings": {
			bindings: []clusterviewv1alpha1.ClusterBinding{
				{
					Cluster:    "prod-cluster",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
				{
					Cluster:    "dev-cluster",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"team-a", "team-b"},
				},
				{
					Cluster:    "staging-cluster",
					Scope:      clusterviewv1alpha1.BindingScopeNamespace,
					Namespaces: []string{"preview"},
				},
				{
					Cluster:    "test-cluster",
					Scope:      clusterviewv1alpha1.BindingScopeCluster,
					Namespaces: []string{"*"},
				},
			},
			expectedClusters:       []string{"prod-cluster", "test-cluster"},
			expectedNamespaceCount: 3,
			checkNamespaces: func(t *testing.T, namespaces []string) {
				assert.Contains(t, namespaces, "dev-cluster:team-a")
				assert.Contains(t, namespaces, "dev-cluster:team-b")
				assert.Contains(t, namespaces, "staging-cluster:preview")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			scope := ConvertBindingsToSimpleAccessScope(tc.bindings)

			// Verify SimpleAccessScope structure
			require.NotNil(t, scope)
			assert.NotEmpty(t, scope.GetId(), "SimpleAccessScope should have a generated ID")
			require.NotNil(t, scope.GetRules())

			// Verify included clusters
			actualClusters := set.NewStringSet(scope.GetRules().GetIncludedClusters()...)
			expectedClusters := set.NewStringSet(tc.expectedClusters...)
			assert.Equal(t, expectedClusters, actualClusters,
				"Expected clusters %v, got %v", tc.expectedClusters, scope.GetRules().GetIncludedClusters())

			// Verify namespace count
			assert.Len(t, scope.GetRules().GetIncludedNamespaces(), tc.expectedNamespaceCount,
				"Expected %d namespaces, got %d", tc.expectedNamespaceCount, len(scope.GetRules().GetIncludedNamespaces()))

			// Additional namespace checks if provided
			if tc.checkNamespaces != nil && len(scope.GetRules().GetIncludedNamespaces()) > 0 {
				// Convert to cluster:namespace format for easier checking
				namespaceStrs := make([]string, len(scope.GetRules().GetIncludedNamespaces()))
				for i, ns := range scope.GetRules().GetIncludedNamespaces() {
					namespaceStrs[i] = ns.GetClusterName() + ":" + ns.GetNamespaceName()
				}
				tc.checkNamespaces(t, namespaceStrs)
			}
		})
	}
}

func TestConvertBindingsToSimpleAccessScope_NamespaceStructure(t *testing.T) {
	bindings := []clusterviewv1alpha1.ClusterBinding{
		{
			Cluster:    "my-cluster",
			Scope:      clusterviewv1alpha1.BindingScopeNamespace,
			Namespaces: []string{"default", "kube-system"},
		},
	}

	scope := ConvertBindingsToSimpleAccessScope(bindings)

	require.NotNil(t, scope)
	require.NotNil(t, scope.GetRules())
	require.Len(t, scope.GetRules().GetIncludedNamespaces(), 2)

	// Verify the structure of namespace entries
	for _, ns := range scope.GetRules().GetIncludedNamespaces() {
		assert.Equal(t, "my-cluster", ns.GetClusterName(), "ClusterName should be set")
		assert.NotEmpty(t, ns.GetNamespaceName(), "NamespaceName should be set")
		assert.Contains(t, []string{"default", "kube-system"}, ns.GetNamespaceName())
		assert.Empty(t, ns.GetClusterId(), "ClusterId should not be set (we only use ClusterName)")
	}
}

func TestConvertBindingsToSimpleAccessScope_EmptyNamespacesList(t *testing.T) {
	bindings := []clusterviewv1alpha1.ClusterBinding{
		{
			Cluster:    "cluster-a",
			Scope:      clusterviewv1alpha1.BindingScopeNamespace,
			Namespaces: []string{}, // Empty namespace list
		},
	}

	scope := ConvertBindingsToSimpleAccessScope(bindings)

	require.NotNil(t, scope)
	require.NotNil(t, scope.GetRules())
	assert.Empty(t, scope.GetRules().GetIncludedNamespaces(), "Should have no namespace entries for empty namespace list")
}

func TestConvertBindingsToSimpleAccessScope_OnlyClusterScoped(t *testing.T) {
	bindings := []clusterviewv1alpha1.ClusterBinding{
		{
			Cluster:    "cluster-1",
			Scope:      clusterviewv1alpha1.BindingScopeCluster,
			Namespaces: []string{"*"},
		},
		{
			Cluster:    "cluster-2",
			Scope:      clusterviewv1alpha1.BindingScopeCluster,
			Namespaces: []string{"*"},
		},
	}

	scope := ConvertBindingsToSimpleAccessScope(bindings)

	require.NotNil(t, scope)
	require.NotNil(t, scope.GetRules())

	// Should have clusters but no namespaces
	assert.Len(t, scope.GetRules().GetIncludedClusters(), 2)
	assert.Empty(t, scope.GetRules().GetIncludedNamespaces())

	clusterSet := set.NewStringSet(scope.GetRules().GetIncludedClusters()...)
	assert.True(t, clusterSet.Contains("cluster-1"))
	assert.True(t, clusterSet.Contains("cluster-2"))
}

func TestConvertBindingsToSimpleAccessScope_OnlyNamespaceScoped(t *testing.T) {
	bindings := []clusterviewv1alpha1.ClusterBinding{
		{
			Cluster:    "cluster-1",
			Scope:      clusterviewv1alpha1.BindingScopeNamespace,
			Namespaces: []string{"ns-1", "ns-2"},
		},
		{
			Cluster:    "cluster-2",
			Scope:      clusterviewv1alpha1.BindingScopeNamespace,
			Namespaces: []string{"ns-3"},
		},
	}

	scope := ConvertBindingsToSimpleAccessScope(bindings)

	require.NotNil(t, scope)
	require.NotNil(t, scope.GetRules())

	// Should have namespaces but no cluster-wide access
	assert.Empty(t, scope.GetRules().GetIncludedClusters())
	assert.Len(t, scope.GetRules().GetIncludedNamespaces(), 3)

	// Verify all namespaces are present
	nsMap := make(map[string]bool)
	for _, ns := range scope.GetRules().GetIncludedNamespaces() {
		key := ns.GetClusterName() + ":" + ns.GetNamespaceName()
		nsMap[key] = true
	}

	assert.True(t, nsMap["cluster-1:ns-1"])
	assert.True(t, nsMap["cluster-1:ns-2"])
	assert.True(t, nsMap["cluster-2:ns-3"])
}
