package tests

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

const (
	resourceAlert        = permissions.Resource("Alert")
	resourceCluster      = permissions.Resource("Cluster")
	resourceConfig       = permissions.Resource("Config")
	resourceDeployment   = permissions.Resource("Deployment")
	resourceImage        = permissions.Resource("Image")
	resourceInstallation = permissions.Resource("InstallationInfo")
	resourceNetworkGraph = permissions.Resource("NetworkGraph")
	resourceNode         = permissions.Resource("Node")
	resourceRisk         = permissions.Resource("Risk")

	clusterClusterID = "clusterID"
	clusterCluster1  = "cluster1"
	clusterMyCluster = "mycluster"

	nsNamespace1 = "namespace1"
	nsNamespace2 = "namespace2"
	nsFoo        = "foo"
	nsBar        = "bar"
	nsBaz        = "baz"
	nsFar        = "far"
)

type testScopeCheckerCoreTestSuite struct {
	suite.Suite
}

func TestTestScopeCheckerCore(t *testing.T) {
	suite.Run(t, new(testScopeCheckerCoreTestSuite))
}

type testScopeCheckerCoreTestCase struct {
	name                string
	scopeCheckerBuilder func(t *testing.T) sac.ScopeCheckerCore
	scopeKeys           []sac.ScopeKey
	tryResults          []sac.TryAllowedResult
}

func (s *testScopeCheckerCoreTestSuite) TestFullMapTestScopeCheckerHierarchyTryAllowed() {
	testcases := []testScopeCheckerCoreTestCase{
		{
			name:                "Read multiple resources some with namespace scope allows read to namespace in restricted scope",
			scopeCheckerBuilder: createTestReadMultipleResourcesSomeWithNamespaceScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceDeployment),
				sac.ClusterScopeKey(clusterClusterID),
				sac.NamespaceScopeKey(nsNamespace2),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny, sac.Deny, sac.Allow},
		},
		{
			name:                "Read multiple resources some with namespace scope denies read to namespace out of scope",
			scopeCheckerBuilder: createTestReadMultipleResourcesSomeWithNamespaceScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceDeployment),
				sac.ClusterScopeKey(clusterClusterID),
				sac.NamespaceScopeKey(nsNamespace1),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:                "Read multiple resources some with namespace scope denies write to namespace from read scope",
			scopeCheckerBuilder: createTestReadMultipleResourcesSomeWithNamespaceScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKey(resourceDeployment),
				sac.ClusterScopeKey(clusterClusterID),
				sac.NamespaceScopeKey(nsNamespace2),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:                "Read multiple resources some with namespace scope allows read to namespace in any allowed-resource-wide scope",
			scopeCheckerBuilder: createTestReadMultipleResourcesSomeWithNamespaceScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceNode),
				sac.ClusterScopeKey(clusterCluster1),
				sac.NamespaceScopeKey(nsNamespace2),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Allow, sac.Allow, sac.Allow},
		},
		{
			name:                "Read multiple resources some with namespace scope allows read to fully included resource",
			scopeCheckerBuilder: createTestReadMultipleResourcesSomeWithNamespaceScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceNode),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Allow},
		},
		{
			name:                "Read from allowed namespace is allowed",
			scopeCheckerBuilder: createTestReadMultipleResourcesWithDifferentNamespaceScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceNetworkGraph),
				sac.ClusterScopeKey(clusterMyCluster),
				sac.NamespaceScopeKey(nsFar),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny, sac.Deny, sac.Allow},
		},
		{
			name:                "Read from excluded namespace is denied",
			scopeCheckerBuilder: createTestReadMultipleResourcesWithDifferentNamespaceScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceNetworkGraph),
				sac.ClusterScopeKey(clusterMyCluster),
				sac.NamespaceScopeKey(nsBar),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:                "Read from excluded resource is denied",
			scopeCheckerBuilder: createTestResourceLevelReadAndReadWriteMixScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceNetworkGraph),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:                "Read from included resource is allowed",
			scopeCheckerBuilder: createTestResourceLevelReadAndReadWriteMixScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceAlert),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Allow},
		},
		{
			name:                "Write to excluded resource is denied",
			scopeCheckerBuilder: createTestResourceLevelReadAndReadWriteMixScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKey(resourceConfig),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:                "Write to included resource is allowed",
			scopeCheckerBuilder: createTestResourceLevelReadAndReadWriteMixScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceDeployment),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Allow},
		},
		{
			name:                "Read from included internal resource is denied",
			scopeCheckerBuilder: createTestResourceLevelReadAndReadWriteMixScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resourceInstallation),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Allow},
		},
		{
			name:                "Write to NOT included internal resource is denied",
			scopeCheckerBuilder: createTestResourceLevelReadAndReadWriteMixScope,
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKey(resourceInstallation),
			},
			tryResults: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny},
		},
	}

	for ix := range testcases {
		tc := testcases[ix]
		s.Run(tc.name, func() {
			scc := tc.scopeCheckerBuilder(s.T())
			globalResult := scc.TryAllowed()
			s.Equal(len(tc.scopeKeys)+1, len(tc.tryResults))
			s.Equal(tc.tryResults[0], globalResult)
			for keyIx := range tc.scopeKeys {
				scc = scc.SubScopeChecker(tc.scopeKeys[keyIx])
				tryResult := scc.TryAllowed()
				s.Equal(tc.tryResults[keyIx+1], tryResult)
			}
		})
	}
}

func createTestReadMultipleResourcesSomeWithNamespaceScope(t *testing.T) sac.ScopeCheckerCore {
	testScope := map[storage.Access]map[permissions.Resource]*sac.TestResourceScope{
		storage.Access_READ_ACCESS: {
			permissions.Resource(resourceCluster): &sac.TestResourceScope{Included: true},
			permissions.Resource(resourceNode):    &sac.TestResourceScope{Included: true},
			permissions.Resource(resourceDeployment): &sac.TestResourceScope{
				Included: false,
				Clusters: map[string]*sac.TestClusterScope{
					clusterClusterID: {
						Included:   false,
						Namespaces: []string{nsNamespace2},
					},
				},
			},
		},
	}
	return sac.TestScopeCheckerCoreFromFullScopeMap(t, testScope)
}

func createTestReadMultipleResourcesWithDifferentNamespaceScope(t *testing.T) sac.ScopeCheckerCore {
	testScope := map[storage.Access]map[permissions.Resource]*sac.TestResourceScope{
		storage.Access_READ_ACCESS: {
			permissions.Resource(resourceDeployment): &sac.TestResourceScope{
				Included: false,
				Clusters: map[string]*sac.TestClusterScope{
					clusterMyCluster: {
						Included:   false,
						Namespaces: []string{nsFoo, nsBar, nsBaz},
					},
				},
			},
			permissions.Resource(resourceNetworkGraph): &sac.TestResourceScope{
				Included: false,
				Clusters: map[string]*sac.TestClusterScope{
					clusterMyCluster: {
						Included:   false,
						Namespaces: []string{nsFoo, nsBaz, nsFar},
					},
				},
			},
		},
	}
	return sac.TestScopeCheckerCoreFromFullScopeMap(t, testScope)
}

func createTestResourceLevelReadAndReadWriteMixScope(t *testing.T) sac.ScopeCheckerCore {
	resourcesWithAccess := []permissions.ResourceWithAccess{
		resourceWithAccess(storage.Access_READ_ACCESS, resourceAlert),
		resourceWithAccess(storage.Access_READ_ACCESS, resourceConfig),
		resourceWithAccess(storage.Access_READ_ACCESS, resourceDeployment),
		resourceWithAccess(storage.Access_READ_ACCESS, resourceImage),
		resourceWithAccess(storage.Access_READ_ACCESS, resourceInstallation),
		resourceWithAccess(storage.Access_READ_ACCESS, resourceRisk),
		resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resourceAlert),
		resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resourceDeployment),
		resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resourceImage),
		resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resourceRisk),
	}
	return sac.TestScopeCheckerCoreFromAccessResourceMap(t, resourcesWithAccess)
}
