package tests

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	. "github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stretchr/testify/assert"
)

func TestAllowFixedScopes(t *testing.T) {
	t.Parallel()

	resA := permissions.Resource("resA")
	resB := permissions.Resource("resB")
	resC := permissions.Resource("resC")

	sc := NewScopeChecker(AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		ResourceScopeKeys(resA, resB),
	))

	cases := []struct {
		scope    []ScopeKey
		expected TryAllowedResult
	}{
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resC),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resC),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resA),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resA),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resB),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resB),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resC),
				ClusterScopeKey("someCluster"),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resC),
				ClusterScopeKey("someCluster"),
			},
			expected: Deny,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resA),
				ClusterScopeKey("someCluster"),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resA),
				ClusterScopeKey("someCluster"),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_ACCESS),
				ResourceScopeKey(resB),
				ClusterScopeKey("someCluster"),
			},
			expected: Allow,
		},
		{
			scope: []ScopeKey{
				AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				ResourceScopeKey(resB),
				ClusterScopeKey("someCluster"),
			},
			expected: Allow,
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, sc.TryAllowed(c.scope...), "expected result for scope %v to be %s", c.scope, c.expected)
	}
}

func TestAllowFixedScopesEffectiveAccessScope(t *testing.T) {
	resA := permissions.Resource("resourceA")
	resB := permissions.Resource("resourceB")
	cluster1 := "cluster1"
	namespaceA := "namespaceA"
	namespaceB := "namespaceB"

	emptyAllowedScope := AllowFixedScopes()

	readAllAllowedScope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS))

	readWriteAllAllowedScope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS))

	readResourceAScope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS),
		ResourceScopeKeys(resA))

	readWriteResourceAScope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		ResourceScopeKeys(resA))

	readResourceACluster1Scope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS),
		ResourceScopeKeys(resA),
		ClusterScopeKeys(cluster1))

	readResourceACluster1NamespacesABScope := AllowFixedScopes(
		AccessModeScopeKeys(storage.Access_READ_ACCESS),
		ResourceScopeKeys(resA),
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
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case write A)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case read B)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for empty fixed scope is Unrestricted for any resource and access (case write B)",
			checker:        emptyAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Unrestricted for read on any resource (case read A)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Denied for write on any resource (case write A)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Unrestricted for read on any resource (case read B)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read fixed scope is Denied for write on any resource (case write B)",
			checker:        readAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case read A)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case write A)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case read B)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write fixed scope is Unrestricted for any resource and access (case write B)",
			checker:        readWriteAllAllowedScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is Unrestricted for read on the resource (case read A)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is denied for write on the resource (case write A)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is denied for read on other resource (case read B)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource fixed scope is denied for write on other resource (case write B)",
			checker:        readResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is Unrestricted for read on the resource (case read A)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is Unrestricted for write on the resource (case write A)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resA),
			expectedScope:  effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is denied for read on other resource (case read B)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read-write resource fixed scope is denied for write on other resource (case write B)",
			checker:        readWriteResourceAScope,
			targetResource: resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resB),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource cluster fixed scope is cluster scope for read on the resource",
			checker:        readResourceACluster1Scope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA),
			expectedScope:  effectiveaccessscope.FromClustersAndNamespacesMap([]string{cluster1}, nil),
		},
		{
			name:           "EAS for read resource cluster fixed scope is denied for other resources",
			checker:        readResourceACluster1Scope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:           "EAS for read resource cluster namespaces fixed scope is cluster namespaces scope for read on the resource",
			checker:        readResourceACluster1NamespacesABScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resA),
			expectedScope: effectiveaccessscope.FromClustersAndNamespacesMap(nil, map[string][]string{
				cluster1: {namespaceA, namespaceB},
			}),
		},
		{
			name:           "EAS for read resource cluster namespaces fixed scope is denied for other resources",
			checker:        readResourceACluster1NamespacesABScope,
			targetResource: resourceWithAccess(storage.Access_READ_ACCESS, resB),
			expectedScope:  effectiveaccessscope.DenyAllEffectiveAccessScope(),
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
