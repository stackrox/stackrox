package tests

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	. "github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
)

func TestAllowFixedScopes(t *testing.T) {
	t.Parallel()

	resA := permissions.ResourceMetadata{
		Resource: permissions.Resource("resA"),
	}
	resB := permissions.ResourceMetadata{
		Resource: permissions.Resource("resB"),
		ReplacingResource: &permissions.ResourceMetadata{
			Resource: permissions.Resource("resD"),
		},
	}
	resC := permissions.ResourceMetadata{
		Resource: permissions.Resource("resC"),
	}

	sc := NewScopeChecker(
		AllowFixedResourceLevelScopes(
			AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			ResourceScopeKeys(resA, resB),
		))

	cases := []struct {
		scope    []ScopeKey
		expected bool
	}{
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
			},
			expected: false,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
			},
			expected: false,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resC.GetResource()),
			},
			expected: false,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resC.GetResource()),
			},
			expected: false,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resA.GetResource()),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resA.GetResource()),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resB.GetResource()),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resB.GetResource()),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resC.GetResource()),
				ClusterScopeKey("someCluster"),
			},
			expected: false,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resC.GetResource()),
				ClusterScopeKey("someCluster"),
			},
			expected: false,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resA.GetResource()),
				ClusterScopeKey("someCluster"),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resA.GetResource()),
				ClusterScopeKey("someCluster"),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resB.GetResource()),
				ClusterScopeKey("someCluster"),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resB.GetResource()),
				ClusterScopeKey("someCluster"),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(*resB.GetReplacingResource()),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(*resB.GetReplacingResource()),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(*resB.GetReplacingResource()),
				ClusterScopeKey("someCluster"),
			},
			expected: true,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(*resB.GetReplacingResource()),
				ClusterScopeKey("someCluster"),
			},
			expected: true,
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, sc.IsAllowed(c.scope...), "expected result for scope %v to be %s", c.scope, c.expected)
	}
}

func TestAllowFixedScopesEffectiveAccessScope(t *testing.T) {
	resA := resources.RegisterDeprecatedResourceMetadataForTest(
		t,
		"resourceA",
		permissions.NamespaceScope,
		permissions.ResourceMetadata{
			Resource: permissions.Resource("resourceC"),
			Scope:    permissions.NamespaceScope,
		},
	)
	resB := resources.RegisterResourceMetadataForTest(
		t,
		"resourceB",
		permissions.GlobalScope,
	)
	resD := resources.RegisterResourceMetadataForTest(
		t,
		"resourceD",
		permissions.ClusterScope,
	)
	cluster1 := "cluster1"
	namespaceA := "namespaceA"
	namespaceB := "namespaceB"

	emptyAllowedScope := AllowFixedGlobalLevelScopes()

	readAllAllowedScope := AllowFixedAccessLevelScopes(
		AccessModeScopeKeyList(storage.Access_READ_ACCESS))

	readWriteAllAllowedScope := AllowFixedAccessLevelScopes(
		AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS))

	readResourceAScope := AllowFixedResourceLevelScopes(
		AccessModeScopeKeyList(storage.Access_READ_ACCESS),
		ResourceScopeKeys(resA))

	readWriteResourceAScope := AllowFixedResourceLevelScopes(
		AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		ResourceScopeKeys(resA))

	readResourceACluster1Scope := AllowFixedClusterLevelScopes(
		AccessModeScopeKeyList(storage.Access_READ_ACCESS),
		ResourceScopeKeys(resA),
		ClusterScopeKeys(cluster1))

	readResourceACluster1NamespacesABScope := AllowFixedNamespaceLevelScopes(
		AccessModeScopeKeyList(storage.Access_READ_ACCESS),
		ResourceScopeKeys(resA),
		ClusterScopeKeys(cluster1),
		NamespaceScopeKeys(namespaceA, namespaceB))

	readResourceDScope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS),
		ResourceScopeKeys(resD))

	readWriteResourceDScope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		ResourceScopeKeys(resD))

	readResourceDCluster1Scope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS),
		ResourceScopeKeys(resD),
		ClusterScopeKeys(cluster1))

	readResourceDCluster1NamespacesABScope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS),
		ResourceScopeKeys(resD),
		ClusterScopeKeys(cluster1),
		NamespaceScopeKeys(namespaceA, namespaceB))

	type testCase struct {
		name           string
		checker        ScopeCheckerCore
		targetResource permissions.ResourceWithAccess
		expectedScope  *effectiveaccessscope.ScopeTree
		expectsError   bool
	}

	testCases := []testCase{
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case read A)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case write A)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case read B)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case write B)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Unrestricted for read on any resource (case read A)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Denied for write on any resource (case write A)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Unrestricted for read on any resource (case read B)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Denied for write on any resource (case write B)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case read A)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case write A)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case read B)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case write B)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is Unrestricted for read on the resource (case read A)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is denied for write on the resource (case write A)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is denied for read on other resource (case read B)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is denied for write on other resource (case write B)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is Unrestricted for read on the resource (case read A)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is Unrestricted for write on the resource (case write A)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is denied for read on other resource (case read B)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is denied for write on other resource (case write B)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource cluster fixed scope is cluster scope for read on the resource",
			checker:        readResourceACluster1Scope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA.GetResource()),
			expectedScope:  effectiveaccessscope.FromClustersAndNamespacesMap([]string{cluster1}, nil),
		},
		{
			name:           "EAS for read resource cluster fixed scope is denied for other resources",
			checker:        readResourceACluster1Scope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource cluster namespaces fixed scope is cluster namespaces scope for read on the resource",
			checker:        readResourceACluster1NamespacesABScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA.GetResource()),
			expectedScope: effectiveaccessscope.FromClustersAndNamespacesMap(nil, map[string][]string{
				cluster1: {namespaceA, namespaceB},
			}),
		},
		{
			name:           "EAS for read resource cluster namespaces fixed scope is denied for other resources",
			checker:        readResourceACluster1NamespacesABScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case read D)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case write D)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Unrestricted for read on any resource (case read D)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Denied for write on any resource (case write D)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case read D)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case write D)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope (cluster) is Unrestricted for read on the resource (case read D)",
			checker:        readResourceDScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope (cluster) is denied for write on the resource (case write D)",
			checker:        readResourceDScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope (cluster) is denied for read on other resource (case read B)",
			checker:        readResourceDScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope (cluster) is denied for write on other resource (case write B)",
			checker:        readResourceDScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope (cluster) is Unrestricted for read on the resource (case read D)",
			checker:        readWriteResourceDScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope (cluster) is Unrestricted for write on the resource (case write D)",
			checker:        readWriteResourceDScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope (cluster) is denied for read on other resource (case read B)",
			checker:        readWriteResourceDScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope (cluster) is denied for write on other resource (case write B)",
			checker:        readWriteResourceDScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource cluster fixed scope (cluster) is cluster scope for read on the resource",
			checker:        readResourceDCluster1Scope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resD.GetResource()),
			expectedScope:  effectiveaccessscope.FromClustersAndNamespacesMap([]string{cluster1}, nil),
		},
		{
			name:           "EAS for read resource cluster fixed scope (cluster) is denied for other resources",
			checker:        readResourceDCluster1Scope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource cluster namespaces fixed scope (cluster) is cluster namespaces scope for read on the resource",
			checker:        readResourceDCluster1NamespacesABScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resD.GetResource()),
			expectedScope: effectiveaccessscope.FromClustersAndNamespacesMap(nil, map[string][]string{
				cluster1: {namespaceA, namespaceB},
			}),
		},
		{
			name:           "EAS for read resource cluster namespaces fixed scope (cluster) is denied for other resources",
			checker:        readResourceDCluster1NamespacesABScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB.GetResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case read C)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case write C)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Unrestricted for read on any resource (case read C)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Denied for write on any resource (case write C)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case read C)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case write C)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is Unrestricted for read on the resource (case read C)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is denied for write on the resource (case write C)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is Unrestricted for read on the resource (case read C)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is Unrestricted for write on the resource (case write C)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, *resA.GetReplacingResource()),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
	}
	for ix := range testCases {
		tc := testCases[ix]
		eas, err := tc.checker.EffectiveAccessScope(tc.targetResource)
		assert.True(t, tc.expectsError == (err != nil))
		assert.NotNil(t, eas)
		assert.Equalf(t, tc.expectedScope.State, eas.State, "Mismatch of Effective Access Scope root state for case %s", tc.name)
		compactActualEAS := eas.Compactify()
		compactExpectedEAS := tc.expectedScope.Compactify()
		assert.Equalf(t, len(compactExpectedEAS), len(compactActualEAS), "Mismatch in number of clusters in scope for case %s", tc.name)
		for cluster := range compactExpectedEAS {
			expectedNamespaces := compactExpectedEAS[cluster]
			actualNamespaces := compactActualEAS[cluster]
			assert.ElementsMatchf(t, expectedNamespaces, actualNamespaces, "Mismatch between namespaces for cluster %s (case %s)", cluster, tc.name)
		}
	}
}
