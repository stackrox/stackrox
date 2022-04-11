package testutils

import (
	"context"

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
	MixedClusterAndNamespaceReadCtx  = "MixedClusterAndNamespaceReadCtx"
)

// GetNamespaceScopedTestContexts provides a set of pre-defined scoped contexts for use in scoped access control tests
func GetNamespaceScopedTestContexts(ctx context.Context, resource permissions.Resource) map[string]context.Context {
	contextMap := make(map[string]context.Context, 0)

	contextMap[UnrestrictedReadCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resource)))

	contextMap[UnrestrictedReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource)))

	contextMap[Cluster1ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster1)))

	contextMap[Cluster1NamespaceAReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceA)))

	contextMap[Cluster1NamespaceBReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceB)))

	contextMap[Cluster1NamespaceCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceC)))

	contextMap[Cluster1NamespacesABReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceB)))

	contextMap[Cluster1NamespacesACReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceC)))

	contextMap[Cluster1NamespacesBCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster1),
				sac.NamespaceScopeKeys(testconsts.NamespaceB, testconsts.NamespaceC)))

	contextMap[Cluster2ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster2)))

	contextMap[Cluster2NamespaceAReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceA)))

	contextMap[Cluster2NamespaceBReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceB)))

	contextMap[Cluster2NamespaceCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceC)))

	contextMap[Cluster2NamespacesABReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceB)))

	contextMap[Cluster2NamespacesACReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceA, testconsts.NamespaceC)))

	contextMap[Cluster2NamespacesBCReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster2),
				sac.NamespaceScopeKeys(testconsts.NamespaceB, testconsts.NamespaceC)))

	contextMap[Cluster3ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resource),
				sac.ClusterScopeKeys(testconsts.Cluster3)))

	contextMap[MixedClusterAndNamespaceReadCtx] =
		sac.WithGlobalAccessScopeChecker(ctx,
			sac.OneStepSCC{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.OneStepSCC{
					sac.ResourceScopeKey(resource): sac.OneStepSCC{
						sac.ClusterScopeKey(testconsts.Cluster1): sac.AllowFixedScopes(sac.NamespaceScopeKeys(testconsts.NamespaceA)),
						sac.ClusterScopeKey(testconsts.Cluster2): sac.AllowAllAccessScopeChecker(),
						sac.ClusterScopeKey(testconsts.Cluster3): sac.AllowFixedScopes(sac.NamespaceScopeKeys(testconsts.NamespaceC)),
					},
				},
			})

	return contextMap
}
