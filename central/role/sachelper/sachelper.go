package sachelper

import (
	"context"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// ClusterSacHelper is an interface to query the scope of the requester for basic cluster information (ID and name).
//
//go:generate mockgen-wrapper
type ClusterSacHelper interface {
	GetClustersForPermissions(
		ctx context.Context,
		requestedPermissions []string,
		pagination *v1.Pagination,
	) ([]*v1.ScopeObject, error)

	IsClusterVisibleForPermissions(
		ctx context.Context,
		clusterID string,
		resourcesWithAccess []permissions.ResourceWithAccess,
	) (bool, error)
}

// ClusterNamespaceSacHelper is an interface to query the scope of the requester for basic cluster and namespace information (ID and name).
//
//go:generate mockgen-wrapper
type ClusterNamespaceSacHelper interface {
	GetNamespacesForClusterAndPermissions(
		ctx context.Context,
		clusterID string,
		requestedPermissions []string,
	) ([]*v1.ScopeObject, error)
}

type clusterSACHelperImpl struct {
	clusterDataStore clusterDS.DataStore
}

type clusterNamespaceSACHelperImpl struct {
	clusterHelper      *clusterSACHelperImpl
	namespaceDataStore namespaceDS.DataStore
}

// NewClusterSacHelper returns a helper object to get information from user scope.
func NewClusterSacHelper(clusterDataStore clusterDS.DataStore) ClusterSacHelper {
	return &clusterSACHelperImpl{
		clusterDataStore: clusterDataStore,
	}
}

// NewClusterNamespaceSacHelper returns a helper object to get information from user scope.
func NewClusterNamespaceSacHelper(clusterDataStore clusterDS.DataStore, namespaceDataStore namespaceDS.DataStore) ClusterNamespaceSacHelper {
	return &clusterNamespaceSACHelperImpl{
		clusterHelper: &clusterSACHelperImpl{
			clusterDataStore: clusterDataStore,
		},
		namespaceDataStore: namespaceDataStore,
	}
}

func (h *clusterSACHelperImpl) GetClustersForPermissions(
	ctx context.Context,
	requestedPermissions []string,
	pagination *v1.Pagination,
) ([]*v1.ScopeObject, error) {
	resourcesWithAccess := listReadPermissions(requestedPermissions, permissions.ClusterScope)
	clusterIDsInScope, hasFullAccess, err := listClusterIDsInScope(ctx, resourcesWithAccess)
	if err != nil {
		return nil, err
	}
	if len(clusterIDsInScope) == 0 && !hasFullAccess {
		return nil, nil
	}

	// Use an elevated context to fetch cluster names associated with the listed IDs.
	// This context must not be propagated.
	// The search is restricted to the cluster name field, and to the clusters allowed
	// by the extended scope.
	var clusterLookupCtx context.Context
	if hasFullAccess {
		clusterLookupCtx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
			),
		)
	} else {
		clusterLookupCtx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
				sac.ClusterScopeKeys(clusterIDsInScope.AsSlice()...),
			),
		)
	}
	query := search.NewQueryBuilder().
		AddStringsHighlighted(search.Cluster, search.WildcardString).
		ProtoQuery()

	allowedSortFields := set.NewStringSet(search.ClusterID.String(), search.Cluster.String())
	query.Pagination = getSanitizedPagination(pagination, allowedSortFields)
	results, err := h.clusterDataStore.Search(clusterLookupCtx, query)
	if err != nil {
		return nil, err
	}

	optionsMap := getClustersOptionsMap()
	clusters := extractScopeElements(results, optionsMap, search.Cluster.String())

	return clusters, nil
}

func (h *clusterSACHelperImpl) IsClusterVisibleForPermissions(
	ctx context.Context,
	clusterID string,
	resourcesWithAccess []permissions.ResourceWithAccess,
) (bool, error) {
	clusterFound, hasFullAccess, err := hasClusterIDInScope(ctx, clusterID, resourcesWithAccess)
	if err != nil {
		return false, err
	}
	if hasFullAccess {
		// Use an elevated context to check the existence of the cluster associated with the listed ID.
		// This context must not be propagated.
		elevatedCtx := sac.WithGlobalAccessScopeChecker(
			ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
			),
		)
		return h.clusterDataStore.Exists(elevatedCtx, clusterID)
	}
	return clusterFound, nil
}

func (h *clusterNamespaceSACHelperImpl) GetNamespacesForClusterAndPermissions(
	ctx context.Context,
	clusterID string,
	requestedPermissions []string,
) ([]*v1.ScopeObject, error) {
	resourcesWithAccess := listReadPermissions(requestedPermissions, permissions.NamespaceScope)
	allNsResourcesWithAccess := listReadPermissions([]string{}, permissions.NamespaceScope)

	clusterVisible, err := h.clusterHelper.IsClusterVisibleForPermissions(ctx, clusterID, allNsResourcesWithAccess)
	if err != nil {
		return nil, err
	}
	if !clusterVisible {
		return nil, errox.NotFound
	}

	namespacesInScope, hasFullAccess, err := listNamespaceNamesInScope(ctx, clusterID, resourcesWithAccess)
	if err != nil {
		return nil, err
	}
	if len(namespacesInScope) == 0 && !hasFullAccess {
		return nil, nil
	}

	// Use an elevated context to fetch namespace IDs and names associated with the listed namespace names.
	// This context must not be propagated.
	// The search is restricted to the namespace name field, and to the namespaces allowed
	// by the extended scope.
	var namespaceLookupCtx context.Context
	if hasFullAccess {
		namespaceLookupCtx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Namespace),
				sac.ClusterScopeKeys(clusterID),
			),
		)
	} else {
		namespaceLookupCtx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Namespace),
				sac.ClusterScopeKeys(clusterID),
				sac.NamespaceScopeKeys(namespacesInScope.AsSlice()...),
			),
		)
	}
	query := search.NewQueryBuilder().
		AddStringsHighlighted(search.Namespace, search.WildcardString).
		ProtoQuery()
	/*
		// Currently, the namespace search overrides pagination information. As a consequence, pagination is disabled here.
		allowedSortFields := set.NewStringSet(search.NamespaceID.String(), search.Namespace.String())
		query.Pagination = getSanitizedPagination(req.GetPagination(), allowedSortFields)
	*/
	results, err := h.namespaceDataStore.Search(namespaceLookupCtx, query)
	if err != nil {
		return nil, err
	}

	optionsMap := getNamespacesOptionsMap()
	namespaces := extractScopeElements(results, optionsMap, search.Namespace.String())
	return namespaces, nil
}
