package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Role)): {
			"/v1.RoleService/GetRoles",
			"/v1.RoleService/GetRole",
		},
		user.With(permissions.Modify(resources.Role)): {
			"/v1.RoleService/CreateRole",
			"/v1.RoleService/SetDefaultRole",
			"/v1.RoleService/UpdateRole",
			"/v1.RoleService/DeleteRole",
		},
	})
)

type serviceImpl struct {
	roleStore store.Store
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

func (s *serviceImpl) GetRoles(context.Context, *v1.Empty) (*v1.GetRolesResponse, error) {
	roles, err := s.roleStore.GetAllRoles()
	if err != nil {
		return nil, err
	}
	return &v1.GetRolesResponse{Roles: roles}, nil
}

func (s *serviceImpl) GetRole(ctx context.Context, id *v1.ResourceByID) (*storage.Role, error) {
	role, err := s.roleStore.GetRole(id.GetId())
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, status.Errorf(codes.NotFound, "Role %s not found", id.GetId())
	}
	return role, nil
}

func (s *serviceImpl) CreateRole(ctx context.Context, role *storage.Role) (*v1.Empty, error) {
	if role.GetGlobalAccess() != storage.Access_NO_ACCESS {
		return nil, status.Errorf(codes.InvalidArgument, "Setting global access is not supported.")
	}
	err := s.roleStore.AddRole(role)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) UpdateRole(ctx context.Context, role *storage.Role) (*v1.Empty, error) {
	if role.GetGlobalAccess() != storage.Access_NO_ACCESS {
		return nil, status.Errorf(codes.InvalidArgument, "Setting global access is not supported.")
	}
	err := s.roleStore.UpdateRole(role)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteRole(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	err := s.roleStore.RemoveRole(id.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}
