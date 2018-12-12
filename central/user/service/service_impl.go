package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/user/store"
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
		user.With(permissions.View(resources.User)): {
			"/v1.UserService/GetUsers",
			"/v1.UserService/GetUser",
		},
		user.With(permissions.Modify(resources.User)): {
			"/v1.UserService/CreateUser",
			"/v1.UserService/UpdateUser",
			"/v1.UserService/DeleteUser",
		},
	})
)

type serviceImpl struct {
	userStore store.Store
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterUserServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterUserServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetUsers(context.Context, *v1.Empty) (*v1.GetUsersResponse, error) {
	users, err := s.userStore.GetAllUsers()
	if err != nil {
		return nil, err
	}
	resp := &v1.GetUsersResponse{
		Users: users,
	}
	return resp, nil
}

func (s *serviceImpl) GetUser(ctx context.Context, id *v1.ResourceByID) (*storage.User, error) {
	user, err := s.userStore.GetUser(id.GetId())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %s not found", id.GetId())
	}
	return user, nil
}
