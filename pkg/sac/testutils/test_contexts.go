package testutils

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	sacresources "github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
)

// Keys to use the pre-defined scopes provided with GetNamespaceScopedTestContexts
const (
	NoAccessCtx                      = "NoAccessCtx"
	OtherResourceReadCtx             = "OtherResourceReadCtx"
	OtherResourceReadWriteCtx        = "OtherResourceReadWriteCtx"
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
	Cluster4ReadWriteCtx             = "Cluster4ReadWriteCtx"
	MixedClusterAndNamespaceReadCtx  = "MixedClusterAndNamespaceReadCtx"
)

// GetNamespaceScopedTestContexts provides a set of pre-defined scoped contexts for use in scoped access control tests
func GetNamespaceScopedTestContexts(ctx context.Context, t *testing.T, resources ...permissions.ResourceMetadata) map[string]context.Context {
	contextMap := make(map[string]context.Context, 0)
	resourceHandles := make([]permissions.ResourceHandle, 0, len(resources))
	for _, r := range resources {
		resourceHandles = append(resourceHandles, r)
	}

	contextMap[NoAccessCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.DenyAllAccessScopeChecker())

	contextMap[OtherResourceReadCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(sacresources.Administration)))

	contextMap[OtherResourceReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(sacresources.Administration)))

	contextMap[UnrestrictedReadCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...)))

	contextMap[UnrestrictedReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...)))

	contextMap[Cluster1ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster1)))

	contextMap[Cluster1NamespaceAReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceA)))

	contextMap[Cluster1NamespaceBReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceB)))

	contextMap[Cluster1NamespaceCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceC)))

	contextMap[Cluster1NamespacesABReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceB)))

	contextMap[Cluster1NamespacesACReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceC)))

	contextMap[Cluster1NamespacesBCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceB, testconsts.NamespaceC)))

	contextMap[Cluster2ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster2)))

	contextMap[Cluster2NamespaceAReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceA)))

	contextMap[Cluster2NamespaceBReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceB)))

	contextMap[Cluster2NamespaceCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceC)))

	contextMap[Cluster2NamespacesABReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceB)))

	contextMap[Cluster2NamespacesACReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceC)))

	contextMap[Cluster2NamespacesBCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceB, testconsts.NamespaceC)))

	contextMap[Cluster3ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster3)))

	contextMap[Cluster3NamespaceAReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster3),
				sac.NamespaceScopeKeys(testconsts.NamespaceA)))

	contextMap[Cluster3NamespaceBReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster3),
				sac.NamespaceScopeKeys(testconsts.NamespaceB)))

	contextMap[Cluster3NamespaceCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster3),
				sac.NamespaceScopeKeys(testconsts.NamespaceC)))

	contextMap[Cluster3NamespacesABReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster3),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceB)))

	contextMap[Cluster3NamespacesACReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster3),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceC)))

	contextMap[Cluster3NamespacesBCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster3),
				sac.NamespaceScopeKeys(testconsts.NamespaceB, testconsts.NamespaceC)))

	contextMap[Cluster4ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster4)))

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

// GetGloballyScopedTestContexts provides a set of pre-defined globally scoped contexts for use in scoped access control tests.
// This function is suitable for testing resources that are globally scoped (not cluster or namespace scoped).
// The otherResource parameter is used to create contexts with access to a different resource than the one being tested.
func GetGloballyScopedTestContexts(ctx context.Context, t *testing.T, otherResource permissions.ResourceMetadata, resources ...permissions.ResourceMetadata) map[string]context.Context {
	contextMap := make(map[string]context.Context, 0)
	resourceHandles := make([]permissions.ResourceHandle, 0, len(resources))
	for _, r := range resources {
		resourceHandles = append(resourceHandles, r)
	}

	contextMap[NoAccessCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.DenyAllAccessScopeChecker())

	contextMap[OtherResourceReadCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(otherResource)))

	contextMap[OtherResourceReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(otherResource)))

	contextMap[UnrestrictedReadCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...)))

	contextMap[UnrestrictedReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...)))

	return contextMap
}
