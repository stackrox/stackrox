package sac

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	targetResource = resources.Deployment
	otherResource  = resources.Namespace

	baseCtx = context.Background()

	noAccessCtx = WithNoAccess(baseCtx)

	fullAccessCtx = WithAllAccess(baseCtx)

	globalTargetReadCtx = WithGlobalAccessScopeChecker(
		baseCtx,
		AllowFixedScopes(
			AccessModeScopeKeys(storage.Access_READ_ACCESS),
			ResourceScopeKeys(targetResource),
		),
	)

	globalTargetReadWriteCtx = WithGlobalAccessScopeChecker(
		baseCtx,
		AllowFixedScopes(
			AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			ResourceScopeKeys(targetResource),
		),
	)

	restrictedTargetReadCtx = WithGlobalAccessScopeChecker(
		baseCtx,
		AllowFixedScopes(
			AccessModeScopeKeys(storage.Access_READ_ACCESS),
			ResourceScopeKeys(targetResource),
			ClusterScopeKeys(fixtureconsts.Cluster1),
		),
	)

	restrictedTargetReadWriteCtx = WithGlobalAccessScopeChecker(
		baseCtx,
		AllowFixedScopes(
			AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			ResourceScopeKeys(targetResource),
			ClusterScopeKeys(fixtureconsts.Cluster1),
		),
	)

	globalOtherReadCtx = WithGlobalAccessScopeChecker(
		baseCtx,
		AllowFixedScopes(
			AccessModeScopeKeys(storage.Access_READ_ACCESS),
			ResourceScopeKeys(otherResource),
		),
	)

	globalOtherReadWriteCtx = WithGlobalAccessScopeChecker(
		baseCtx,
		AllowFixedScopes(
			AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			ResourceScopeKeys(otherResource),
		),
	)

	restrictedOtherReadCtx = WithGlobalAccessScopeChecker(
		baseCtx,
		AllowFixedScopes(
			AccessModeScopeKeys(storage.Access_READ_ACCESS),
			ResourceScopeKeys(otherResource),
			ClusterScopeKeys(fixtureconsts.Cluster1),
		),
	)

	restrictedOtherReadWriteCtx = WithGlobalAccessScopeChecker(
		baseCtx,
		AllowFixedScopes(
			AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			ResourceScopeKeys(otherResource),
			ClusterScopeKeys(fixtureconsts.Cluster1),
		),
	)

	clusterEarthName   = "Earth"
	clusterArrakisName = "Arrakis"

	clusterEarth = &storage.Cluster{
		Id:   fixtureconsts.Cluster1,
		Name: clusterEarthName,
	}

	clusterArrakis = &storage.Cluster{
		Id:   fixtureconsts.Cluster2,
		Name: clusterArrakisName,
	}

	namespaceSkunkWorks = &storage.NamespaceMetadata{
		Id:          uuid.NewTestUUID(1).String(),
		Name:        nsSkunkWorks,
		ClusterId:   clusterEarth.Id,
		ClusterName: clusterEarth.Name,
	}

	namespaceFraunhofer = &storage.NamespaceMetadata{
		Id:          uuid.NewTestUUID(2).String(),
		Name:        nsFraunhofer,
		ClusterId:   clusterEarth.Id,
		ClusterName: clusterEarth.Name,
	}

	namespaceCERN = &storage.NamespaceMetadata{
		Id:          uuid.NewTestUUID(3).String(),
		Name:        nsCERN,
		ClusterId:   clusterEarth.Id,
		ClusterName: clusterEarth.Name,
	}

	namespaceJPL = &storage.NamespaceMetadata{
		Id:          uuid.NewTestUUID(4).String(),
		Name:        nsJPL,
		ClusterId:   clusterEarth.Id,
		ClusterName: clusterEarth.Name,
	}

	namespaceAtreides = &storage.NamespaceMetadata{
		Id:          uuid.NewTestUUID(11).String(),
		Name:        nsAtreides,
		ClusterId:   clusterArrakis.Id,
		ClusterName: clusterArrakis.Name,
	}

	namespaceBeneGesserit = &storage.NamespaceMetadata{
		Id:          uuid.NewTestUUID(12).String(),
		Name:        nsBeneGesserit,
		ClusterId:   clusterArrakis.Id,
		ClusterName: clusterArrakis.Name,
	}

	namespaceFremen = &storage.NamespaceMetadata{
		Id:          uuid.NewTestUUID(13).String(),
		Name:        nsFremen,
		ClusterId:   clusterArrakis.Id,
		ClusterName: clusterArrakis.Name,
	}

	namespaceHarkonnen = &storage.NamespaceMetadata{
		Id:          uuid.NewTestUUID(14).String(),
		Name:        nsHarkonnen,
		ClusterId:   clusterArrakis.Id,
		ClusterName: clusterArrakis.Name,
	}

	namespaceSpacingGuild = &storage.NamespaceMetadata{
		Id:          uuid.NewTestUUID(15).String(),
		Name:        nsSpacingGuild,
		ClusterId:   clusterArrakis.Id,
		ClusterName: clusterArrakis.Name,
	}

	allNamespaces = []*storage.NamespaceMetadata{
		namespaceSkunkWorks,
		namespaceFraunhofer,
		namespaceCERN,
		namespaceJPL,
		namespaceAtreides,
		namespaceBeneGesserit,
		namespaceFremen,
		namespaceHarkonnen,
		namespaceSpacingGuild,
	}
)

func TestHasGlobalRead(t *testing.T) {
	helper := ForResource(targetResource)

	for name, tc := range map[string]struct {
		ctx             context.Context
		expectedAllowed bool
	}{
		"No access context does not have global read": {
			ctx:             noAccessCtx,
			expectedAllowed: false,
		},
		"Full access context has global read": {
			ctx:             fullAccessCtx,
			expectedAllowed: true,
		},
		"Target resource full read context has global read": {
			ctx:             globalTargetReadCtx,
			expectedAllowed: true,
		},
		"Target resource full read write context has global read": {
			ctx:             globalTargetReadWriteCtx,
			expectedAllowed: true,
		},
		"Target resource cluster-level read context does not have global read": {
			ctx:             restrictedTargetReadCtx,
			expectedAllowed: false,
		},
		"Target resource cluster-level read-write context does not have global read": {
			ctx:             restrictedTargetReadWriteCtx,
			expectedAllowed: false,
		},
		"Other resource full read context does not have global read": {
			ctx:             globalOtherReadCtx,
			expectedAllowed: false,
		},
		"Other resource full read-write context does not have global read": {
			ctx:             globalOtherReadWriteCtx,
			expectedAllowed: false,
		},
		"Other resource cluster-level read context does not have global read": {
			ctx:             restrictedOtherReadCtx,
			expectedAllowed: false,
		},
		"Other resource cluster-level read-write context does not have global read": {
			ctx:             restrictedOtherReadWriteCtx,
			expectedAllowed: false,
		},
	} {
		t.Run(name, func(it *testing.T) {
			hasFullRead, err := helper.HasGlobalRead(tc.ctx)
			assert.NoError(it, err)
			assert.Equal(it, tc.expectedAllowed, hasFullRead)
		})
	}
}

func TestHasGlobalWrite(t *testing.T) {
	helper := ForResource(targetResource)

	for name, tc := range map[string]struct {
		ctx             context.Context
		expectedAllowed bool
	}{
		"No access context does not have global read": {
			ctx:             noAccessCtx,
			expectedAllowed: false,
		},
		"Full access context has global read": {
			ctx:             fullAccessCtx,
			expectedAllowed: true,
		},
		"Target resource full read context has global read": {
			ctx:             globalTargetReadCtx,
			expectedAllowed: false,
		},
		"Target resource full read write context has global read": {
			ctx:             globalTargetReadWriteCtx,
			expectedAllowed: true,
		},
		"Target resource cluster-level read context does not have global read": {
			ctx:             restrictedTargetReadCtx,
			expectedAllowed: false,
		},
		"Target resource cluster-level read-write context does not have global read": {
			ctx:             restrictedTargetReadWriteCtx,
			expectedAllowed: false,
		},
		"Other resource full read context does not have global read": {
			ctx:             globalOtherReadCtx,
			expectedAllowed: false,
		},
		"Other resource full read-write context does not have global read": {
			ctx:             globalOtherReadWriteCtx,
			expectedAllowed: false,
		},
		"Other resource cluster-level read context does not have global read": {
			ctx:             restrictedOtherReadCtx,
			expectedAllowed: false,
		},
		"Other resource cluster-level read-write context does not have global read": {
			ctx:             restrictedOtherReadWriteCtx,
			expectedAllowed: false,
		},
	} {
		t.Run(name, func(it *testing.T) {
			hasFullWrite, err := helper.HasGlobalWrite(tc.ctx)
			assert.NoError(it, err)
			assert.Equal(it, tc.expectedAllowed, hasFullWrite)
		})
	}
}

func TestFilterAccessibleNamespacesForRead(t *testing.T) {
	for name, tc := range map[string]struct {
		ctx                        context.Context
		targetResource             permissions.ResourceMetadata
		expectedFilteredNamespaces []*storage.NamespaceMetadata
	}{
		"Full Access context will let all namespaces through for a global-scoped resource": {
			ctx:                        fullAccessCtx,
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: allNamespaces,
		},
		"Full Access context will let all namespaces through for a cluster-scoped resource": {
			ctx:                        fullAccessCtx,
			targetResource:             resources.Node,
			expectedFilteredNamespaces: allNamespaces,
		},
		"Full Access context will let all namespaces through for a namespace-scoped resource": {
			ctx:                        fullAccessCtx,
			targetResource:             resources.Image,
			expectedFilteredNamespaces: allNamespaces,
		},
		"No Access context will let no namespace through for a global-scoped resource": {
			ctx:                        noAccessCtx,
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"No Access context will let no namespace through for a cluster-scoped resource": {
			ctx:                        noAccessCtx,
			targetResource:             resources.Node,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"No Access context will let no namespace through for a namespace-scoped resource": {
			ctx:                        noAccessCtx,
			targetResource:             resources.Image,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"Cluster-restricted context will let all namespaces through for a global-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Administration),
					ClusterScopeKeys(clusterArrakis.Id),
				),
			),
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: allNamespaces,
		},
		"Cluster-restricted context will only let the namespaces within that cluster through for a cluster-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Node),
					ClusterScopeKeys(clusterArrakis.Id),
				),
			),
			targetResource: resources.Node,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{
				namespaceAtreides,
				namespaceBeneGesserit,
				namespaceFremen,
				namespaceHarkonnen,
				namespaceSpacingGuild,
			},
		},
		"Cluster-restricted context will only let the namespaces within that cluster through for a namespace-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Image),
					ClusterScopeKeys(clusterEarth.Id),
				),
			),
			targetResource: resources.Image,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{
				namespaceSkunkWorks,
				namespaceFraunhofer,
				namespaceCERN,
				namespaceJPL,
			},
		},
		"Namespace-restricted context will let all namespaces through for a global-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Administration),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: allNamespaces,
		},
		"Namespace-restricted context will let the namespaces within the same cluster through for cluster-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Node),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource: resources.Node,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{
				namespaceAtreides,
				namespaceBeneGesserit,
				namespaceFremen,
				namespaceHarkonnen,
				namespaceSpacingGuild,
			},
		},
		"Namespace-restricted context will only let the defined namespaces through for namespace-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Image),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource: resources.Image,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{
				namespaceAtreides,
				namespaceBeneGesserit,
				namespaceFremen,
			},
		},
		"Namespace-restricted context will let no namespace through for another resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Image),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource:             resources.Deployment,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
	} {
		t.Run(name, func(it *testing.T) {
			helper := ForResource(tc.targetResource)
			filteredNamespaces, err := helper.FilterAccessibleNamespaces(tc.ctx, storage.Access_READ_ACCESS, allNamespaces)
			assert.NoError(it, err)
			protoassert.ElementsMatch(it, filteredNamespaces, tc.expectedFilteredNamespaces)
		})
	}
}

func TestFilterAccessibleNamespacesForWrite(t *testing.T) {
	for name, tc := range map[string]struct {
		ctx                        context.Context
		targetResource             permissions.ResourceMetadata
		expectedFilteredNamespaces []*storage.NamespaceMetadata
	}{
		"Full Access context will let all namespaces through for a global-scoped resource": {
			ctx:                        fullAccessCtx,
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: allNamespaces,
		},
		"Full Access context will let all namespaces through for a cluster-scoped resource": {
			ctx:                        fullAccessCtx,
			targetResource:             resources.Node,
			expectedFilteredNamespaces: allNamespaces,
		},
		"Full Access context will let all namespaces through for a namespace-scoped resource": {
			ctx:                        fullAccessCtx,
			targetResource:             resources.Image,
			expectedFilteredNamespaces: allNamespaces,
		},
		"No Access context will let no namespace through for a global-scoped resource": {
			ctx:                        noAccessCtx,
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"No Access context will let no namespace through for a cluster-scoped resource": {
			ctx:                        noAccessCtx,
			targetResource:             resources.Node,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"No Access context will let no namespace through for a namespace-scoped resource": {
			ctx:                        noAccessCtx,
			targetResource:             resources.Image,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"Cluster-restricted read context will let no namespaces through for a global-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Administration),
					ClusterScopeKeys(clusterArrakis.Id),
				),
			),
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"Cluster-restricted write context will let all namespaces through for a global-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					ResourceScopeKeys(resources.Administration),
					ClusterScopeKeys(clusterArrakis.Id),
				),
			),
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: allNamespaces,
		},
		"Cluster-restricted read context will let no namespaces through for a cluster-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Node),
					ClusterScopeKeys(clusterArrakis.Id),
				),
			),
			targetResource:             resources.Node,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"Cluster-restricted write context will only let the namespaces within that cluster through for a cluster-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					ResourceScopeKeys(resources.Node),
					ClusterScopeKeys(clusterArrakis.Id),
				),
			),
			targetResource: resources.Node,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{
				namespaceAtreides,
				namespaceBeneGesserit,
				namespaceFremen,
				namespaceHarkonnen,
				namespaceSpacingGuild,
			},
		},
		"Cluster-restricted read context will let no namespace through for a namespace-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Image),
					ClusterScopeKeys(clusterEarth.Id),
				),
			),
			targetResource:             resources.Image,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"Cluster-restricted write context will only let the namespaces within that cluster through for a namespace-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					ResourceScopeKeys(resources.Image),
					ClusterScopeKeys(clusterEarth.Id),
				),
			),
			targetResource: resources.Image,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{
				namespaceSkunkWorks,
				namespaceFraunhofer,
				namespaceCERN,
				namespaceJPL,
			},
		},
		"Namespace-restricted read context will let no namespace through for a global-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Administration),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"Namespace-restricted write context will let all namespaces through for a global-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					ResourceScopeKeys(resources.Administration),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource:             resources.Administration,
			expectedFilteredNamespaces: allNamespaces,
		},
		"Namespace-restricted read context will let no namespace through for cluster-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Node),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource:             resources.Node,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"Namespace-restricted write context will let the namespaces within the same cluster through for cluster-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					ResourceScopeKeys(resources.Node),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource: resources.Node,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{
				namespaceAtreides,
				namespaceBeneGesserit,
				namespaceFremen,
				namespaceHarkonnen,
				namespaceSpacingGuild,
			},
		},
		"Namespace-restricted read context will let no namespace through for namespace-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS),
					ResourceScopeKeys(resources.Image),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource:             resources.Image,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
		"Namespace-restricted write context will only let the defined namespaces through for namespace-scoped resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					ResourceScopeKeys(resources.Image),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource: resources.Image,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{
				namespaceAtreides,
				namespaceBeneGesserit,
				namespaceFremen,
			},
		},
		"Namespace-restricted context will let no namespace through for another resource": {
			ctx: WithGlobalAccessScopeChecker(
				context.Background(),
				AllowFixedScopes(
					AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					ResourceScopeKeys(resources.Image),
					ClusterScopeKeys(clusterArrakis.Id),
					NamespaceScopeKeys(nsAtreides, nsBeneGesserit, nsFremen),
				),
			),
			targetResource:             resources.Deployment,
			expectedFilteredNamespaces: []*storage.NamespaceMetadata{},
		},
	} {
		t.Run(name, func(it *testing.T) {
			helper := ForResource(tc.targetResource)
			filteredNamespaces, err := helper.FilterAccessibleNamespaces(tc.ctx, storage.Access_READ_WRITE_ACCESS, allNamespaces)
			assert.NoError(it, err)
			protoassert.ElementsMatch(it, filteredNamespaces, tc.expectedFilteredNamespaces)
		})
	}
}
