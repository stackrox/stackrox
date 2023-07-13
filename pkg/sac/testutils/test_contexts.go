package testutils

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
)

// Keys to use the pre-defined scopes provided with GetNamespaceScopedTestContexts
const (
	UnrestrictedReadCtx              = "UnrestrictedReadCtx"
	UnrestrictedReadWriteCtx         = "UnrestrictedReadWriteCtx"
	Cluster1ReadWriteCtx             = "Cluster1ReadWriteCtx"
	Cluster1NamespaceAReadWriteCtx   = "Cluster1NamespaceAReadWriteCtx"
	Cluster1NamespaceBReadWriteCtx   = "Cluster1NamespaceBReadWriteCtx"
	Cluster1NamespaceCReadWriteCtx   = "Cluster1NamespaceCReadWriteCtx"
	Cluster1NamespacesABReadWriteCtx = "Cluster1NamespacesABReadWriteCtx"
	Cluster1NamespacesACReadWriteCtx = "Cluster1NamespacesACReadWriteCtx"
	Cluster1NamespacesBCReadWriteCtx = "Cluster1NamespacesBCReadWriteCtx"
	Cluster2ReadWriteCtx             = "Cluster2ReadWriteCtx"
	Cluster2NamespaceAReadWriteCtx   = "Cluster2NamespaceAReadWriteCtx"
	Cluster2NamespaceBReadWriteCtx   = "Cluster2NamespaceBReadWriteCtx"
	Cluster2NamespaceCReadWriteCtx   = "Cluster2NamespaceCReadWriteCtx"
	Cluster2NamespacesABReadWriteCtx = "Cluster2NamespacesABReadWriteCtx"
	Cluster2NamespacesACReadWriteCtx = "Cluster2NamespacesACReadWriteCtx"
	Cluster2NamespacesBCReadWriteCtx = "Cluster2NamespacesBCReadWriteCtx"
	Cluster3ReadWriteCtx             = "Cluster3ReadWriteCtx"
	Cluster3NamespaceAReadWriteCtx   = "Cluster3NamespaceAReadWriteCtx"
	Cluster3NamespaceBReadWriteCtx   = "Cluster3NamespaceBReadWriteCtx"
	Cluster3NamespaceCReadWriteCtx   = "Cluster3NamespaceCReadWriteCtx"
	Cluster3NamespacesABReadWriteCtx = "Cluster3NamespacesABReadWriteCtx"
	Cluster3NamespacesACReadWriteCtx = "Cluster3NamespacesACReadWriteCtx"
	Cluster3NamespacesBCReadWriteCtx = "Cluster3NamespacesBCReadWriteCtx"
	MixedClusterAndNamespaceReadCtx  = "MixedClusterAndNamespaceReadCtx"
)

// GetNamespaceScopedTestContexts provides a set of pre-defined scoped contexts for use in scoped access control tests
func GetNamespaceScopedTestContexts(ctx context.Context, t *testing.T, resources ...permissions.ResourceMetadata) map[string]context.Context {
	contextMap := make(map[string]context.Context, 0)
	resourceHandles := make([]permissions.ResourceHandle, 0, len(resources))
	for _, r := range resources {
		resourceHandles = append(resourceHandles, r)
	}

	contextMap[UnrestrictedReadCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedResourceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...)))

	contextMap[UnrestrictedReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedResourceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...)))

	contextMap[Cluster1ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedClusterLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster1)))

	contextMap[Cluster1NamespaceAReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster1),
				sac.NamespaceScopeKeyList(testconsts.NamespaceA)))

	contextMap[Cluster1NamespaceBReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster1),
				sac.NamespaceScopeKeyList(testconsts.NamespaceB)))

	contextMap[Cluster1NamespaceCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster1),
				sac.NamespaceScopeKeyList(testconsts.NamespaceC)))

	contextMap[Cluster1NamespacesABReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster1),
				sac.NamespaceScopeKeyList(testconsts.NamespaceA, testconsts.NamespaceB)))

	contextMap[Cluster1NamespacesACReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster1),
				sac.NamespaceScopeKeyList(testconsts.NamespaceA, testconsts.NamespaceC)))

	contextMap[Cluster1NamespacesBCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster1),
				sac.NamespaceScopeKeyList(testconsts.NamespaceB, testconsts.NamespaceC)))

	contextMap[Cluster2ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedClusterLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster2)))

	contextMap[Cluster2NamespaceAReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster2),
				sac.NamespaceScopeKeyList(testconsts.NamespaceA)))

	contextMap[Cluster2NamespaceBReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster2),
				sac.NamespaceScopeKeyList(testconsts.NamespaceB)))

	contextMap[Cluster2NamespaceCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster2),
				sac.NamespaceScopeKeyList(testconsts.NamespaceC)))

	contextMap[Cluster2NamespacesABReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster2),
				sac.NamespaceScopeKeyList(testconsts.NamespaceA, testconsts.NamespaceB)))

	contextMap[Cluster2NamespacesACReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster2),
				sac.NamespaceScopeKeyList(testconsts.NamespaceA, testconsts.NamespaceC)))

	contextMap[Cluster2NamespacesBCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster2),
				sac.NamespaceScopeKeyList(testconsts.NamespaceB, testconsts.NamespaceC)))

	contextMap[Cluster3ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedClusterLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster3)))

	contextMap[Cluster3NamespaceAReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster3),
				sac.NamespaceScopeKeyList(testconsts.NamespaceA)))

	contextMap[Cluster3NamespaceBReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster3),
				sac.NamespaceScopeKeyList(testconsts.NamespaceB)))

	contextMap[Cluster3NamespaceCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster3),
				sac.NamespaceScopeKeyList(testconsts.NamespaceC)))

	contextMap[Cluster3NamespacesABReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster3),
				sac.NamespaceScopeKeyList(testconsts.NamespaceA, testconsts.NamespaceB)))

	contextMap[Cluster3NamespacesACReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster3),
				sac.NamespaceScopeKeyList(testconsts.NamespaceA, testconsts.NamespaceC)))

	contextMap[Cluster3NamespacesBCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedNamespaceLevelScopes(
				sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeyList(resourceHandles...),
				sac.ClusterScopeKeyList(testconsts.Cluster3),
				sac.NamespaceScopeKeyList(testconsts.NamespaceB, testconsts.NamespaceC)))

	mixedResourceScope := &sac.TestResourceScope{
		Clusters: map[string]*sac.TestClusterScope{
			testconsts.Cluster1: {Namespaces: []string{testconsts.NamespaceA}},
			testconsts.Cluster2: {Included: true},
			testconsts.Cluster3: {Namespaces: []string{testconsts.NamespaceC}},
		},
	}
	mixedAccessScope := map[permissions.Resource]*sac.TestResourceScope{}
	for _, r := range resources {
		mixedAccessScope[r.GetResource()] = mixedResourceScope
		if r.GetReplacingResource() != nil {
			mixedAccessScope[*r.GetReplacingResource()] = mixedResourceScope
		}
	}
	contextMap[MixedClusterAndNamespaceReadCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.TestScopeCheckerCoreFromFullScopeMap(t,
				sac.TestScopeMap{
					storage.Access_READ_ACCESS: mixedAccessScope,
				}))

	return contextMap
}
