package service

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/role/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
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

type serviceImpl struct {
	roleDataStore datastore.DataStore
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
	for _, role := range roles {
		utils.FillAccessList(role)
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
	utils.FillAccessList(role)
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
	utils.FillAccessList(role)
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
		grpcCode := errorTypeToGrpcCode(err)
		return nil, status.Errorf(grpcCode, "failed to retrieve access scope %q: %v", id.GetId(), err)
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
		grpcCode := errorTypeToGrpcCode(err)
		return nil, status.Errorf(grpcCode, "failed to retrieve access scopes: %v", err)
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
	scope.Id = utils.AccessScopeIDPrefix + uuid.NewV4().String()

	// Store the augmented access scope; report back on error. Note the access
	// scope is referenced by its name because that's what the caller knows.
	err := s.roleDataStore.AddAccessScope(ctx, scope)
	if err != nil {
		grpcCode := errorTypeToGrpcCode(err)
		return nil, status.Errorf(grpcCode, "failed to store access scope %q: %v", scope.GetName(), err)
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
		grpcCode := errorTypeToGrpcCode(err)
		return nil, status.Errorf(grpcCode, "failed to update access scope %q: %v", scope.GetId(), err)
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteSimpleAccessScope(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	if !features.ScopedAccessControl.Enabled() {
		return nil, status.Error(codes.Unimplemented, "feature not enabled")
	}

	err := s.roleDataStore.RemoveAccessScope(ctx, id.GetId())
	if err != nil {
		grpcCode := errorTypeToGrpcCode(err)
		return nil, status.Errorf(grpcCode, "failed to delete access scope %q: %v", id.GetId(), err)
	}

	return &v1.Empty{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Helpers                                                                    //
//                                                                            //

// TODO(ROX-6983): Make this mapping available for all services.
func errorTypeToGrpcCode(err error) codes.Code {
	switch {
	case errors.Is(err, errorhelpers.ErrNotFound):
		return codes.NotFound
	case errors.Is(err, errorhelpers.ErrInvalidArgs):
		return codes.InvalidArgument
	case errors.Is(err, errorhelpers.ErrAlreadyExists):
		return codes.AlreadyExists
	case errors.Is(err, sac.ErrPermissionDenied):
		return codes.PermissionDenied
	default:
		return codes.Internal
	}
}
