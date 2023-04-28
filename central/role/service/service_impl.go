package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	namespaceMappings "github.com/stackrox/rox/central/namespace/index/mappings"
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/accessscope"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		// TODO: ROX-14398 Replace Role with Access
		user.With(permissions.View(resources.Role)): {
			"/v1.RoleService/GetRoles",
			"/v1.RoleService/GetRole",
			"/v1.RoleService/ListPermissionSets",
			"/v1.RoleService/GetPermissionSet",
			"/v1.RoleService/ListSimpleAccessScopes",
			"/v1.RoleService/GetSimpleAccessScope",
		},
		// TODO: ROX-14398 Replace Role with Access
		user.With(permissions.View(resources.Role), permissions.View(resources.Cluster), permissions.View(resources.Namespace)): {
			"/v1.RoleService/ComputeEffectiveAccessScope",
		},
		// TODO: ROX-14398 Replace Role with Access
		user.With(permissions.Modify(resources.Role)): {
			"/v1.RoleService/CreateRole",
			"/v1.RoleService/SetDefaultRole",
			"/v1.RoleService/UpdateRole",
			"/v1.RoleService/DeleteRole",
			"/v1.RoleService/PostPermissionSet",
			"/v1.RoleService/PutPermissionSet",
			"/v1.RoleService/DeletePermissionSet",
			"/v1.RoleService/PostSimpleAccessScope",
			"/v1.RoleService/PutSimpleAccessScope",
			"/v1.RoleService/DeleteSimpleAccessScope",
		},
		user.With(): {
			"/v1.RoleService/GetClustersForPermissions",
			"/v1.RoleService/GetNamespacesForClusterAndPermissions",
		},
		allow.Anonymous(): {
			"/v1.RoleService/GetResources",
			"/v1.RoleService/GetMyPermissions",
		},
	})
)

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	v1.UnimplementedRoleServiceServer

	roleDataStore      datastore.DataStore
	clusterDataStore   clusterDS.DataStore
	namespaceDataStore namespaceDS.DataStore
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterRoleServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterRoleServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetRoles(ctx context.Context, _ *v1.Empty) (*v1.GetRolesResponse, error) {
	roles, err := s.roleDataStore.GetAllRoles(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve roles")
	}

	// List roles in the same order for consistency across requests.
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].GetName() < roles[j].GetName()
	})

	return &v1.GetRolesResponse{Roles: roles}, nil
}

func (s *serviceImpl) GetRole(ctx context.Context, id *v1.ResourceByID) (*storage.Role, error) {
	role, found, err := s.roleDataStore.GetRole(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve role %q", id.GetId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "failed to retrieve role %q", id.GetId())
	}
	return role, nil
}

func (s *serviceImpl) GetMyPermissions(ctx context.Context, _ *v1.Empty) (*v1.GetPermissionsResponse, error) {
	return GetMyPermissions(ctx)
}

func (s *serviceImpl) CreateRole(ctx context.Context, roleRequest *v1.CreateRoleRequest) (*v1.Empty, error) {
	role := roleRequest.GetRole()

	// Check role request correctness.
	if role.GetName() != "" && role.GetName() != roleRequest.GetName() {
		return nil, errox.InvalidArgs.CausedBy("different role names in path and body")
	}
	role.Name = roleRequest.GetName()

	// Empty access scope ID is deprecated. Fill the default during the adoption
	// period.
	// TODO(ROX-9510): remove this block.
	if role.GetAccessScopeId() == "" {
		role.AccessScopeId = rolePkg.AccessScopeIncludeAll.GetId()
	}
	err := s.roleDataStore.AddRole(ctx, role)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) UpdateRole(ctx context.Context, role *storage.Role) (*v1.Empty, error) {
	// Empty access scope ID is deprecated. Fill the default during the adoption
	// period.
	// TODO(ROX-9510): remove this block.
	if role.GetAccessScopeId() == "" {
		role.AccessScopeId = rolePkg.AccessScopeIncludeAll.GetId()
	}
	err := s.roleDataStore.UpdateRole(ctx, role)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteRole(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	err := s.roleDataStore.RemoveRole(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to delete role %q", id.GetId())
	}
	return &v1.Empty{}, nil
}

// GetResources returns all the possible resources in the system
func (s *serviceImpl) GetResources(context.Context, *v1.Empty) (*v1.GetResourcesResponse, error) {
	resourceList := resources.ListAll()
	resources := make([]string, 0, len(resourceList))
	for _, r := range resourceList {
		resources = append(resources, string(r))
	}
	return &v1.GetResourcesResponse{
		Resources: resources,
	}, nil
}

// GetMyPermissions returns the permissions for a user based on the context.
func GetMyPermissions(ctx context.Context) (*v1.GetPermissionsResponse, error) {
	// Get the perms from the current user context.
	id, err := authn.IdentityFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetPermissionsResponse{
		ResourceToAccess: id.Permissions(),
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Permission sets                                                            //
//                                                                            //

func (s *serviceImpl) GetPermissionSet(ctx context.Context, id *v1.ResourceByID) (*storage.PermissionSet, error) {
	permissionSet, found, err := s.roleDataStore.GetPermissionSet(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve permission set %s", id.GetId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "failed to retrieve permission set %s", id.GetId())
	}

	return permissionSet, nil
}

func (s *serviceImpl) ListPermissionSets(ctx context.Context, _ *v1.Empty) (*v1.ListPermissionSetsResponse, error) {
	permissionSets, err := s.roleDataStore.GetAllPermissionSets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve permission sets")
	}

	// List permission sets in the same order for consistency across requests.
	sort.Slice(permissionSets, func(i, j int) bool {
		return permissionSets[i].GetName() < permissionSets[j].GetName()
	})

	return &v1.ListPermissionSetsResponse{PermissionSets: permissionSets}, nil
}

func (s *serviceImpl) PostPermissionSet(ctx context.Context, permissionSet *storage.PermissionSet) (*storage.PermissionSet, error) {
	if permissionSet.GetId() != "" {
		return nil, errox.InvalidArgs.CausedBy("setting id field is not allowed")
	}
	permissionSet.Id = rolePkg.GeneratePermissionSetID()

	// Store the augmented permission set; report back on error. Note the
	// permission set is referenced by its name because that's what the caller
	// knows.
	err := s.roleDataStore.AddPermissionSet(ctx, permissionSet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to store permission set %q", permissionSet.GetName())
	}

	// Assume AddPermissionSet() does not make modifications to the protobuf.
	return permissionSet, nil
}

func (s *serviceImpl) PutPermissionSet(ctx context.Context, permissionSet *storage.PermissionSet) (*v1.Empty, error) {
	err := s.roleDataStore.UpdatePermissionSet(ctx, permissionSet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update permission set %s", permissionSet.GetId())
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeletePermissionSet(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	err := s.roleDataStore.RemovePermissionSet(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to delete permission set %s", id.GetId())
	}

	return &v1.Empty{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Access scopes                                                              //
//                                                                            //

func (s *serviceImpl) GetSimpleAccessScope(ctx context.Context, id *v1.ResourceByID) (*storage.SimpleAccessScope, error) {
	scope, found, err := s.roleDataStore.GetAccessScope(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve access scope %s", id.GetId())
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "failed to retrieve access scope %s", id.GetId())
	}

	return scope, nil
}

func (s *serviceImpl) ListSimpleAccessScopes(ctx context.Context, _ *v1.Empty) (*v1.ListSimpleAccessScopesResponse, error) {
	scopes, err := s.roleDataStore.GetAllAccessScopes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve access scopes")
	}

	// List access scopes in the same order for consistency across requests.
	sort.Slice(scopes, func(i, j int) bool {
		return scopes[i].GetName() < scopes[j].GetName()
	})

	return &v1.ListSimpleAccessScopesResponse{AccessScopes: scopes}, nil
}

func (s *serviceImpl) PostSimpleAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) (*storage.SimpleAccessScope, error) {
	if scope.GetId() != "" {
		return nil, errox.InvalidArgs.CausedBy("setting id field is not allowed")
	}
	scope.Id = rolePkg.GenerateAccessScopeID()

	// Store the augmented access scope; report back on error. Note the access
	// scope is referenced by its name because that's what the caller knows.
	err := s.roleDataStore.AddAccessScope(ctx, scope)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to store access scope %q", scope.GetName())
	}

	// Assume AddAccessScope() does not make modifications to the protobuf.
	return scope, nil
}

func (s *serviceImpl) PutSimpleAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) (*v1.Empty, error) {
	err := s.roleDataStore.UpdateAccessScope(ctx, scope)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update access scope %s", scope.GetId())
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteSimpleAccessScope(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	err := s.roleDataStore.RemoveAccessScope(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to delete access scope %s", id.GetId())
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) ComputeEffectiveAccessScope(ctx context.Context, req *v1.ComputeEffectiveAccessScopeRequest) (*storage.EffectiveAccessScope, error) {
	// If we're here, service-level authz has already verified that the caller
	// has at least READ permission on the Role resource.
	err := accessscope.ValidateSimpleAccessScopeRules(req.GetAccessScope().GetSimpleRules())
	if err != nil {
		return nil, errors.Wrap(err, "failed to compute effective access scope")
	}

	// ctx might not have access to all known clusters and namespaces and hence
	// the resulting effective access scope might not include all known scopes,
	//
	// Imagine Alice has write access to Role and read access to scoped Cluster
	// resources. She can create access scopes that will apply to all clusters
	// but while she is creating them she would only see a sliced view.
	readScopesCtx := ctx

	clusters, err := s.clusterDataStore.GetClusters(readScopesCtx)
	if err != nil {
		return nil, errors.Errorf("failed to compute effective access scope: %v", err)
	}

	namespaces, err := s.namespaceDataStore.GetAllNamespaces(readScopesCtx)
	if err != nil {
		return nil, errors.Errorf("failed to compute effective access scope: %v", err)
	}

	response, err := effectiveAccessScopeForSimpleAccessScope(req.GetAccessScope().GetSimpleRules(), clusters, namespaces, req.GetDetail())
	if err != nil {
		return nil, errors.Errorf("failed to compute effective access scope: %v", err)
	}

	return response, nil
}

func (s *serviceImpl) GetClustersForPermissions(ctx context.Context, req *v1.GetClustersForPermissionsRequest) (*v1.GetClustersForPermissionsResponse, error) {
	requestedPermissions := req.GetPermissions()

	resourcesWithAccess := listReadPermissions(requestedPermissions, permissions.ClusterScope)
	clusterIDsInScope, hasFullAccess, err := listClusterIDsInScope(ctx, resourcesWithAccess)
	if err != nil {
		return nil, err
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
	query.Pagination = getSanitizedPagination(req.GetPagination(), allowedSortFields)
	results, err := s.clusterDataStore.Search(clusterLookupCtx, query)
	if err != nil {
		return nil, err
	}

	optionsMap := getClustersOptionsMap()
	clusters := extractScopeElements(results, optionsMap, search.Cluster.String())
	response := &v1.GetClustersForPermissionsResponse{
		Clusters: clusters,
	}
	return response, nil
}

func (s *serviceImpl) GetNamespacesForClusterAndPermissions(ctx context.Context, req *v1.GetNamespaceForClusterAndPermissionsRequest) (*v1.GetNamespacesForClusterAndPermissionsResponse, error) {
	requestedPermissions := req.GetPermissions()
	clusterID := req.GetClusterId()

	resourcesWithAccess := listReadPermissions(requestedPermissions, permissions.NamespaceScope)

	clusterVisible, err := s.isClusterVisibleForRequester(ctx, clusterID)
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
	results, err := s.namespaceDataStore.Search(namespaceLookupCtx, query)
	if err != nil {
		return nil, err
	}

	optionsMap := getNamespacesOptionsMap()
	namespaces := extractScopeElements(results, optionsMap, search.Namespace.String())
	response := &v1.GetNamespacesForClusterAndPermissionsResponse{
		Namespaces: namespaces,
	}
	return response, nil
}

////////////////////////////////////////////////////////////////////////////////
// Helpers                                                                    //
//                                                                            //

// effectiveAccessScopeForSimpleAccessScope computes the effective access scope
// for the given rules and converts it to the desired response.
func effectiveAccessScopeForSimpleAccessScope(scopeRules *storage.SimpleAccessScope_Rules, clusters []*storage.Cluster, namespaces []*storage.NamespaceMetadata, detail v1.ComputeEffectiveAccessScopeRequest_Detail) (*storage.EffectiveAccessScope, error) {
	tree, err := effectiveaccessscope.ComputeEffectiveAccessScope(scopeRules, clusters, namespaces, detail)
	if err != nil {
		return nil, err
	}

	response, err := effectiveaccessscope.ToEffectiveAccessScope(tree)
	if err != nil {
		return nil, err
	}

	return response, nil
}

const (
	hasFullAccess    = true
	hasPartialAccess = false
)

func listReadPermissions(
	requestedPermissions []string,
	scope permissions.ResourceScope,
) []permissions.ResourceWithAccess {
	readPermissions := resources.AllResourcesViewPermissions()
	indexedScopeReadPermissions := make(map[string]permissions.ResourceWithAccess, 0)
	scopeReadPermissions := make([]permissions.ResourceWithAccess, 0, len(readPermissions))
	for _, permission := range readPermissions {
		if permission.Resource.GetScope() >= scope {
			scopeReadPermissions = append(scopeReadPermissions, permission)
			indexedScopeReadPermissions[permission.Resource.String()] = permission
		}
	}
	if len(requestedPermissions) == 0 {
		return scopeReadPermissions
	}
	scopeRequestedReadPermissions := make([]permissions.ResourceWithAccess, 0, len(scopeReadPermissions))
	for _, permission := range sliceutils.Unique(requestedPermissions) {
		if resourceWithAccess, found := indexedScopeReadPermissions[permission]; found {
			scopeRequestedReadPermissions = append(scopeRequestedReadPermissions, resourceWithAccess)
		}
	}
	return scopeRequestedReadPermissions
}

func getRequesterScopeForReadPermission(
	ctx context.Context,
	resourceWithAccess permissions.ResourceWithAccess,
) (*effectiveaccessscope.ScopeTree, error) {
	scopeChecker := sac.ForResource(resourceWithAccess.Resource).ScopeChecker(ctx, storage.Access_READ_ACCESS)
	return scopeChecker.EffectiveAccessScope(resourceWithAccess)
}

// listClusterIDsInScope consolidates the list of cluster IDs in the user scopes associated
// with the requested resources and access level.
// - If one of the allowed scopes is unrestricted, then the string set is returned empty
// and the returned boolean is true.
// - If no allowed scope is unrestricted, the string set contains the cluster IDs allowed by
// the user scopes associated with the requested resources, and the returned boolean is false.
func listClusterIDsInScope(
	ctx context.Context,
	resourcesWithAccess []permissions.ResourceWithAccess,
) (set.StringSet, bool, error) {
	clusterIDsInScope := set.NewStringSet()
	for _, r := range resourcesWithAccess {
		scope, err := getRequesterScopeForReadPermission(ctx, r)
		if err != nil {
			return set.NewStringSet(), hasPartialAccess, err
		}
		if scope == nil || scope.State == effectiveaccessscope.Excluded {
			continue
		}
		if scope.State == effectiveaccessscope.Included {
			return set.NewStringSet(), hasFullAccess, nil
		}
		clusterIDs := scope.GetClusterIDs()
		for _, clusterID := range clusterIDs {
			if clusterNode := scope.GetClusterByID(clusterID); clusterNode != nil &&
				clusterNode.State != effectiveaccessscope.Excluded {
				clusterIDsInScope.Add(clusterID)
			}
		}
	}
	return clusterIDsInScope, hasPartialAccess, nil
}

func getClustersOptionsMap() search.OptionsMap {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return schema.ClustersSchema.OptionsMap
	}
	return clusterMappings.OptionsMap
}

func (s *serviceImpl) isClusterVisibleForRequester(
	ctx context.Context,
	clusterID string,
) (bool, error) {
	resourcesWithAccess := listReadPermissions([]string{}, permissions.ClusterScope)
	clusterIDsInScope, isFullAccess, err := listClusterIDsInScope(ctx, resourcesWithAccess)
	if err != nil {
		return false, err
	}
	if !isFullAccess && clusterIDsInScope.Contains(clusterID) {
		return true, nil
	}
	if isFullAccess {
		clusterLookupElevatedAccess := sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
			),
		)
		exists, err := s.clusterDataStore.Exists(clusterLookupElevatedAccess, clusterID)
		if err != nil {
			return false, err
		}
		return exists, nil
	}
	return false, nil
}

// listNamespaceNamesInScope consolidates the list of names of namespaces in the cluster matching
// the requested ID and allowed by user scopes associated with the requested resources
// and access level.
// - If one of the allowed scopes is unrestricted for the requested cluster, then the string set
// is returned empty and the returned boolean is true.
// - If no allowed scope is unrestricted for the requested cluster, the string set contains
// the names of the namespaces allowed by the user scopes associated with the requested resources,
// and the returned boolean is false.
func listNamespaceNamesInScope(
	ctx context.Context,
	clusterID string,
	resourcesWithAccess []permissions.ResourceWithAccess,
) (set.StringSet, bool, error) {
	noNamespaces := set.NewStringSet()
	namespacesInScope := set.NewStringSet()
	for _, r := range resourcesWithAccess {
		scope, err := getRequesterScopeForReadPermission(ctx, r)
		if err != nil {
			return noNamespaces, hasPartialAccess, err
		}
		if scope == nil || scope.State == effectiveaccessscope.Excluded {
			continue
		}
		if scope.State == effectiveaccessscope.Included {
			return noNamespaces, hasFullAccess, nil
		}
		clusterScope := scope.GetClusterByID(clusterID)
		if clusterScope == nil || clusterScope.State == effectiveaccessscope.Excluded {
			continue
		}
		if clusterScope.State == effectiveaccessscope.Included {
			return noNamespaces, hasFullAccess, nil
		}
		for namespace, namespaceScope := range clusterScope.Namespaces {
			if namespaceScope == nil || namespaceScope.State == effectiveaccessscope.Excluded {
				continue
			}
			namespacesInScope.Add(namespace)
		}
	}
	return namespacesInScope, hasPartialAccess, nil
}

func getSanitizedPagination(requested *v1.Pagination, allowedSortFields set.StringSet) *v1.QueryPagination {
	if requested == nil {
		return nil
	}
	sanitized := &v1.QueryPagination{
		Limit:       requested.GetLimit(),
		Offset:      requested.GetOffset(),
		SortOptions: nil,
	}
	if requested.GetSortOption() != nil {
		sortField := requested.GetSortOption().GetField()
		if allowedSortFields.Contains(sortField) {
			sanitizedSortOption := &v1.QuerySortOption{
				Field:          sortField,
				Reversed:       requested.GetSortOption().GetReversed(),
				SearchAfterOpt: nil,
			}
			sanitized.SortOptions = append(sanitized.SortOptions, sanitizedSortOption)
		}
	}
	return sanitized
}

func getNamespacesOptionsMap() search.OptionsMap {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return schema.NamespacesSchema.OptionsMap
	}
	return namespaceMappings.OptionsMap
}

func extractScopeElements(results []search.Result, optionsMap search.OptionsMap, searchedField string) []*v1.ScopeObject {
	scopeElements := make([]*v1.ScopeObject, 0, len(results))
	targetField, fieldFound := optionsMap.Get(searchedField)
	for _, r := range results {
		objID := r.ID
		objName := ""
		if fieldFound {
			for _, v := range r.Matches[targetField.GetFieldPath()] {
				if len(v) > 0 {
					objName = v
					break
				}
			}
		}
		if len(objName) == 0 {
			objName = fmt.Sprintf("%s with ID %s", searchedField, objID)
		}
		element := &v1.ScopeObject{
			Id:   objID,
			Name: objName,
		}
		scopeElements = append(scopeElements, element)
	}
	return scopeElements
}
