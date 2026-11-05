package effectiveaccessscope

import (
	"slices"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func namespaceInventoryScope(known, included []string) *ScopeTree {
	group := MatchedNamespaceGroup{ClusterName: "cluster", Complete: true}
	for _, name := range known {
		namespace := MatchedNamespace{ID: name + "-id", Name: name}
		if slices.Contains(included, name) {
			group.Namespaces = append(group.Namespaces, namespace)
		} else {
			group.ExcludedNamespaces = append(group.ExcludedNamespaces, namespace)
		}
	}
	return FromMatchingScopes([]Cluster{&storage.Cluster{Id: "cluster-id", Name: "cluster"}}, nil, []MatchedNamespaceGroup{group})
}

func TestNamespaceExclusionMap(t *testing.T) {
	known := []string{"namespace1", "namespace2", "namespace3", "namespace4"}
	tree := namespaceInventoryScope(known, []string{"namespace1", "namespace2", "namespace4"})
	cluster := tree.GetClusterByID("cluster-id")
	require.NotNil(t, cluster)
	require.Equal(t, Partial, cluster.State)
	require.Equal(t, Included, cluster.namespaceDefault)
	require.Len(t, cluster.Namespaces, 1)
	assert.Equal(t, Excluded, cluster.Namespaces["namespace3"].State)
	assert.True(t, cluster.NamespaceAllowed("namespace1"))
	assert.False(t, cluster.NamespaceAllowed("namespace3"))
	assert.False(t, cluster.NamespaceAllowed("unknown"))
	assert.Nil(t, cluster.GetNamespaceByName("unknown"))
	assert.Equal(t, "namespace1-id", cluster.GetNamespaceByName("namespace1").Attributes.ID)
	assert.Equal(t, ScopeTreeCompacted{"cluster": {"namespace1", "namespace2", "namespace4"}}, tree.Compactify())
	response, err := ToEffectiveAccessScope(tree)
	require.NoError(t, err)
	require.Len(t, response.GetClusters(), 1)
	require.Len(t, response.GetClusters()[0].GetNamespaces(), 3)
	for _, namespace := range response.GetClusters()[0].GetNamespaces() {
		assert.Equal(t, storage.EffectiveAccessScope_INCLUDED, namespace.GetState())
		assert.NotEqual(t, "namespace3-id", namespace.GetId())
	}

	all := namespaceInventoryScope(known, known)
	assert.Equal(t, Partial, all.State)
	assert.Equal(t, Partial, all.GetClusterByID("cluster-id").State)
	assert.Empty(t, all.GetClusterByID("cluster-id").Namespaces)
	assert.True(t, all.GetClusterByID("cluster-id").HasNamespaceAccess())
	assert.False(t, all.GetClusterByID("cluster-id").NamespaceAllowed("unknown"))
	assert.Equal(t, Excluded, namespaceInventoryScope(known, nil).State)
}

func TestNamespaceExclusionMerge(t *testing.T) {
	known := []string{"namespace1", "namespace2", "namespace3", "namespace4"}
	except3 := func() *ScopeTree {
		return namespaceInventoryScope(known, []string{"namespace1", "namespace2", "namespace4"})
	}
	except4 := func() *ScopeTree {
		return namespaceInventoryScope(known, []string{"namespace1", "namespace2", "namespace3"})
	}
	namedOnly3 := func() *ScopeTree {
		return FromMatchingScopes([]Cluster{&storage.Cluster{Id: "cluster-id", Name: "cluster"}}, nil,
			[]MatchedNamespaceGroup{{ClusterName: "cluster", Namespaces: []MatchedNamespace{{Name: "namespace3"}}}})
	}
	tests := map[string]struct {
		left, right func() *ScopeTree
		allowed     []string
	}{
		"same exclusion":                {except3, except3, []string{"namespace1", "namespace2", "namespace4"}},
		"different exclusions":          {except3, except4, known},
		"inclusion overrides exclusion": {except3, namedOnly3, known},
		"reverse order":                 {namedOnly3, except3, known},
		"different inventories": {
			func() *ScopeTree {
				return namespaceInventoryScope([]string{"a", "b", "c", "d"}, []string{"a", "b", "c"})
			},
			func() *ScopeTree {
				return namespaceInventoryScope([]string{"b", "c", "d", "e"}, []string{"b", "c", "e"})
			},
			[]string{"a", "b", "c", "e"},
		},
		"full cluster overrides": {except3, func() *ScopeTree {
			return FromMatchingScopes([]Cluster{&storage.Cluster{Id: "cluster-id", Name: "cluster"}}, []string{"cluster-id"}, nil)
		}, nil},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			left, right := test.left(), test.right()
			leftBefore, rightBefore := left.Compactify(), right.Compactify()
			merged := DenyAllEffectiveAccessScope()
			merged.Merge(left)
			merged.Merge(right)
			cluster := merged.GetClusterByID("cluster-id")
			require.NotNil(t, cluster)
			if test.allowed == nil {
				assert.Equal(t, Included, cluster.State)
				assert.True(t, cluster.NamespaceAllowed("unknown"))
			} else {
				assert.ElementsMatch(t, test.allowed, slices.Collect(cluster.AllowedNamespaceNames()))
				assert.False(t, cluster.NamespaceAllowed("unknown"))
			}
			// A merged tree owns its maps, including the inventory snapshot.
			if cluster.namespaceUniverse != nil {
				cluster.namespaceUniverse["mutated"] = "mutated-id"
			}
			for _, namespace := range cluster.Namespaces {
				namespace.State = Included
			}
			cluster.Namespaces = nil
			assert.Equal(t, leftBefore, left.Compactify())
			assert.Equal(t, rightBefore, right.Compactify())
		})
	}
}

func TestNamespaceMergeAllSubsets(t *testing.T) {
	leftKnown, rightKnown := []string{"a", "b", "c"}, []string{"b", "c", "d"}
	for leftMask := range 8 {
		for rightMask := range 8 {
			var left, right []string
			for i, name := range leftKnown {
				if leftMask&(1<<i) != 0 {
					left = append(left, name)
				}
			}
			for i, name := range rightKnown {
				if rightMask&(1<<i) != 0 {
					right = append(right, name)
				}
			}
			tree := namespaceInventoryScope(leftKnown, left)
			tree.Merge(namespaceInventoryScope(rightKnown, right))
			cluster := tree.GetClusterByID("cluster-id")
			for _, name := range []string{"a", "b", "c", "d", "unknown"} {
				allowed := cluster != nil && cluster.NamespaceAllowed(name)
				assert.Equal(t, slices.Contains(left, name) || slices.Contains(right, name), allowed,
					"left=%v right=%v name=%s", left, right, name)
			}
		}
	}
}

func TestNamespaceExclusionDuplicateNames(t *testing.T) {
	tree := FromMatchingScopes([]Cluster{&storage.Cluster{Id: "cluster-id", Name: "cluster"}}, nil, []MatchedNamespaceGroup{{
		ClusterName: "cluster", Complete: true,
		Namespaces: []MatchedNamespace{
			{ID: "2", Name: "duplicate"},
			{ID: "1", Name: "duplicate"},
			{ID: "3", Name: "other"},
		},
		ExcludedNamespaces: []MatchedNamespace{{ID: "0", Name: "duplicate"}, {ID: "4", Name: "excluded"}},
	}})
	cluster := tree.GetClusterByID("cluster-id")
	require.Equal(t, Included, cluster.namespaceDefault)
	assert.True(t, cluster.NamespaceAllowed("duplicate"))
	assert.Equal(t, "1", cluster.GetNamespaceByName("duplicate").Attributes.ID)
	assert.False(t, cluster.NamespaceAllowed("excluded"))
}
