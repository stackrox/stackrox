package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/role/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Role)): {
			"/v1.RoleService/GetRoles",
			"/v1.RoleService/GetRole",
			"/v1.RoleService/ListSimpleAccessScopes",
			"/v1.RoleService/GetSimpleAccessScope",
		},
		user.With(permissions.View(resources.Role), permissions.View(resources.Cluster), permissions.View(resources.Namespace)): {
			"/v1.RoleService/ComputeEffectiveAccessScope",
		},
		user.With(permissions.Modify(resources.Role)): {
			"/v1.RoleService/CreateRole",
			"/v1.RoleService/SetDefaultRole",
			"/v1.RoleService/UpdateRole",
			"/v1.RoleService/DeleteRole",
			"/v1.RoleService/PostSimpleAccessScope",
			"/v1.RoleService/PutSimpleAccessScope",
			"/v1.RoleService/DeleteSimpleAccessScope",
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
	roleDataStore datastore.DataStore

	// TODO(ROX-7076): The built-in authorization plugin is supposed to take
	//   over fetching clusters and namespaces. It would do it in a smarter way
	//   than just going to the datastore for every request.
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
		return nil, err
	}
	return &v1.GetRolesResponse{Roles: roles}, nil
}

func (s *serviceImpl) GetRole(ctx context.Context, id *v1.ResourceByID) (*storage.Role, error) {
	role, err := s.roleDataStore.GetRole(ctx, id.GetId())
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, status.Errorf(codes.NotFound, "Role %s not found", id.GetId())
	}
	return role, nil
}

func (s *serviceImpl) GetMyPermissions(ctx context.Context, _ *v1.Empty) (*storage.Role, error) {
	return GetMyPermissions(ctx)
}

func (s *serviceImpl) CreateRole(ctx context.Context, role *storage.Role) (*v1.Empty, error) {
	if role.GetGlobalAccess() != storage.Access_NO_ACCESS {
		return nil, status.Error(codes.InvalidArgument, "Setting global access is not supported.")
	}
	err := s.roleDataStore.AddRole(ctx, role)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) UpdateRole(ctx context.Context, role *storage.Role) (*v1.Empty, error) {
	if role.GetGlobalAccess() != storage.Access_NO_ACCESS {
		return nil, status.Error(codes.InvalidArgument, "Setting global access is not supported.")
	}
	err := s.roleDataStore.UpdateRole(ctx, role)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteRole(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	role, err := s.roleDataStore.GetRole(ctx, id.GetId())
	if err != nil {
		return nil, err
	} else if role == nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Role '%s' not found", id.GetId()))
	}

	err = s.roleDataStore.RemoveRole(ctx, id.GetId())
	if err != nil {
		return nil, err
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
func GetMyPermissions(ctx context.Context) (*storage.Role, error) {
	// Get the role from the current user context.
	id := authn.IdentityFromContext(ctx)
	if id == nil {
		return nil, status.Error(codes.Internal, "unable to retrieve user identity")
	}
	role := id.Permissions().Clone()
	role.Name = "" // Clear name since this concept can't be applied to a user (Permission may result from many roles).
	return role, nil
}

////////////////////////////////////////////////////////////////////////////////
// Access scopes                                                              //
//                                                                            //

func (s *serviceImpl) GetSimpleAccessScope(ctx context.Context, id *v1.ResourceByID) (*storage.SimpleAccessScope, error) {
	if !features.ScopedAccessControl.Enabled() {
		return nil, status.Error(codes.Unimplemented, "feature not enabled")
	}

	scope, found, err := s.roleDataStore.GetAccessScope(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve access scope %q", id.GetId())
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, "failed to retrieve access scope %q: not found", id.GetId())
	}

	return scope, nil
}

func (s *serviceImpl) ListSimpleAccessScopes(ctx context.Context, _ *v1.Empty) (*v1.ListSimpleAccessScopesResponse, error) {
	if !features.ScopedAccessControl.Enabled() {
		return nil, status.Error(codes.Unimplemented, "feature not enabled")
	}

	scopes, err := s.roleDataStore.GetAllAccessScopes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve access scopes")
	}

	return &v1.ListSimpleAccessScopesResponse{AccessScopes: scopes}, nil
}

func (s *serviceImpl) PostSimpleAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) (*storage.SimpleAccessScope, error) {
	if !features.ScopedAccessControl.Enabled() {
		return nil, status.Error(codes.Unimplemented, "feature not enabled")
	}

	if scope.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "setting id field is not allowed")
	}
	scope.Id = utils.GenerateAccessScopeID()

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
	if !features.ScopedAccessControl.Enabled() {
		return nil, status.Error(codes.Unimplemented, "feature not enabled")
	}

	err := s.roleDataStore.UpdateAccessScope(ctx, scope)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to update access scope %q", scope.GetId())
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteSimpleAccessScope(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	if !features.ScopedAccessControl.Enabled() {
		return nil, status.Error(codes.Unimplemented, "feature not enabled")
	}

	err := s.roleDataStore.RemoveAccessScope(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to delete access scope %q", id.GetId())
	}

	return &v1.Empty{}, nil
}

// TODO(ROX-7076): Instead of fetching all clusters and namespaces for each
//   request, rely on optimizations made by built-in scoped authorizer.
func (s *serviceImpl) ComputeEffectiveAccessScope(ctx context.Context, req *v1.ComputeEffectiveAccessScopeRequest) (*storage.EffectiveAccessScope, error) {
	if !features.ScopedAccessControl.Enabled() {
		return nil, status.Error(codes.Unimplemented, "feature not enabled")
	}

	// If we're here, service-level authz has already verified that the caller
	// has at least READ permission on the Role resource.
	err := utils.ValidateSimpleAccessScopeRules(req.GetAccessScope().GetSimpleRules())
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
		return nil, status.Errorf(codes.Internal, "failed to compute effective access scope: %v", err)
	}

	namespaces, err := s.namespaceDataStore.GetNamespaces(readScopesCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute effective access scope: %v", err)
	}

	response, err := effectiveAccessScopeForSimpleAccessScope(req.GetAccessScope().GetSimpleRules(), clusters, namespaces, req.GetDetail())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute effective access scope: %v", err)
	}

	return response, nil
}

////////////////////////////////////////////////////////////////////////////////
// Helpers                                                                    //
//                                                                            //

type effectiveAccessScopeTreeExtras struct {
	id     string
	name   string
	labels map[string]string
}

// effectiveAccessScopeForSimpleAccessScope computes the effective access scope
// for the given rules and converts it to the desired response.
func effectiveAccessScopeForSimpleAccessScope(scopeRules *storage.SimpleAccessScope_Rules, clusters []*storage.Cluster, namespaces []*storage.NamespaceMetadata, detail v1.ComputeEffectiveAccessScopeRequest_Detail) (*storage.EffectiveAccessScope, error) {
	tree, err := sac.ComputeEffectiveAccessScope(scopeRules, clusters, namespaces)
	if err != nil {
		return nil, err
	}

	augmentEffectiveAccessScopeTreeWithExtras(tree, clusters, namespaces, detail)
	if detail == v1.ComputeEffectiveAccessScopeRequest_MINIMAL {
		compactifyEffectiveAccessScopeTree(tree)
	}
	response, err := convertEffectiveAccessScopeTreeToEffectiveAccessScope(tree)
	if err != nil {
		return nil, err
	}
	sortScopesInEffectiveAccessScope(response)

	return response, nil
}

// augmentEffectiveAccessScopeTreeWithExtras enriches the effective access scope
// tree with scopes' IDs and labels based on the desired detail level.
func augmentEffectiveAccessScopeTreeWithExtras(tree *sac.EffectiveAccessScopeTree, clusters []*storage.Cluster, namespaces []*storage.NamespaceMetadata, detail v1.ComputeEffectiveAccessScopeRequest_Detail) {
	// Augment clusters. Assume cluster name is unique.
	for _, clusterExtra := range clusters {
		cluster, found := tree.Clusters[clusterExtra.GetName()]
		if !found {
			log.Warnf("cluster %q not found in effective access scope tree", clusterExtra.GetName())
			continue
		}

		extras := effectiveAccessScopeTreeExtras{
			id: clusterExtra.GetId(),
		}
		if detail != v1.ComputeEffectiveAccessScopeRequest_MINIMAL {
			extras.name = clusterExtra.GetName()
		}
		if detail == v1.ComputeEffectiveAccessScopeRequest_HIGH {
			extras.labels = clusterExtra.GetLabels()
		}

		cluster.Extras = &extras
	}

	// Augment namespaces. Assume pair <cluster name, namespace name> is unique.
	for _, namespacesExtra := range namespaces {
		cluster, found := tree.Clusters[namespacesExtra.GetClusterName()]
		if !found {
			log.Warnf("cluster %q not found in effective access scope tree", namespacesExtra.GetClusterName())
		}

		namespace, found := cluster.Namespaces[namespacesExtra.GetName()]
		if !found {
			log.Warnf("namespace %q not found in effective access scope tree", namespacesExtra.GetName())
		}

		extras := effectiveAccessScopeTreeExtras{
			id: namespacesExtra.GetId(),
		}
		if detail != v1.ComputeEffectiveAccessScopeRequest_MINIMAL {
			extras.name = namespacesExtra.GetName()
		}
		if detail == v1.ComputeEffectiveAccessScopeRequest_HIGH {
			extras.labels = namespacesExtra.GetLabels()
		}

		namespace.Extras = &extras
	}
}

// compactifyEffectiveAccessScopeTree removes subtrees with roots in the
// Excluded state from the effective access scope tree, as well as subtrees of
// nodes in the Included state. The resulting tree tallies with the MINIMAL
// level of detail for ComputeEffectiveAccessScopeRequest.
func compactifyEffectiveAccessScopeTree(tree *sac.EffectiveAccessScopeTree) {
	for clusterName, clusterSubTree := range tree.Clusters {
		if clusterSubTree.State == sac.Included {
			clusterSubTree.Namespaces = nil
			continue
		}
		if clusterSubTree.State == sac.Excluded {
			delete(tree.Clusters, clusterName)
			continue
		}

		for namespaceName, namespaceSubTree := range clusterSubTree.Namespaces {
			if namespaceSubTree.State == sac.Excluded {
				delete(clusterSubTree.Namespaces, namespaceName)
			}
		}

		if len(clusterSubTree.Namespaces) == 0 {
			delete(tree.Clusters, clusterName)
		}
	}
}

// convertEffectiveAccessScopeTreeToEffectiveAccessScope converts effective
// access scope tree with enriched nodes to storage.EffectiveAccessScope.
func convertEffectiveAccessScopeTreeToEffectiveAccessScope(tree *sac.EffectiveAccessScopeTree) (*storage.EffectiveAccessScope, error) {
	response := &storage.EffectiveAccessScope{}
	if len(tree.Clusters) != 0 {
		response.Clusters = make([]*storage.EffectiveAccessScope_Cluster, 0, len(tree.Clusters))
	}

	for clusterName, clusterSubTree := range tree.Clusters {
		extras, ok := clusterSubTree.Extras.(*effectiveAccessScopeTreeExtras)
		if !ok {
			return nil, errors.Errorf("rich data not available for cluster %q", clusterName)
		}
		cluster := &storage.EffectiveAccessScope_Cluster{
			Id:     extras.id,
			Name:   extras.name,
			State:  convertScopeStateToEffectiveAccessScopeState(clusterSubTree.State),
			Labels: extras.labels,
		}
		if len(clusterSubTree.Namespaces) != 0 {
			cluster.Namespaces = make([]*storage.EffectiveAccessScope_Namespace, 0, len(clusterSubTree.Namespaces))
		}

		for namespaceName, namespaceSubTree := range clusterSubTree.Namespaces {
			extras, ok := namespaceSubTree.Extras.(*effectiveAccessScopeTreeExtras)
			if !ok {
				return nil, errors.Errorf("rich data not available for namespace '%s::%s'", clusterName, namespaceName)
			}
			namespace := &storage.EffectiveAccessScope_Namespace{
				Id:     extras.id,
				Name:   extras.name,
				State:  convertScopeStateToEffectiveAccessScopeState(namespaceSubTree.State),
				Labels: extras.labels,
			}

			cluster.Namespaces = append(cluster.Namespaces, namespace)
		}

		response.Clusters = append(response.Clusters, cluster)
	}

	return response, nil
}

func sortScopesInEffectiveAccessScope(msg *storage.EffectiveAccessScope) {
	clusters := msg.GetClusters()
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].GetId() < clusters[j].GetId()
	})

	for _, cluster := range clusters {
		namespaces := cluster.GetNamespaces()
		sort.Slice(namespaces, func(i, j int) bool {
			return namespaces[i].GetId() < namespaces[j].GetId()
		})
	}
}

func convertScopeStateToEffectiveAccessScopeState(scopeState sac.ScopeState) storage.EffectiveAccessScope_State {
	switch scopeState {
	case sac.Excluded:
		return storage.EffectiveAccessScope_EXCLUDED
	case sac.Partial:
		return storage.EffectiveAccessScope_PARTIAL
	case sac.Included:
		return storage.EffectiveAccessScope_INCLUDED
	default:
		return storage.EffectiveAccessScope_UNKNOWN
	}
}
