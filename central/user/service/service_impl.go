package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/group/datastore/serialize"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/central/user/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stackrox/stackrox/pkg/grpc/authz"
	"github.com/stackrox/stackrox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/stackrox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.User)): {
			"/v1.UserService/GetUsers",
			"/v1.UserService/GetUser",
			"/v1.UserService/GetUsersAttributes",
		},
	})
)

type serviceImpl struct {
	users datastore.DataStore
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

func (s *serviceImpl) GetUsers(ctx context.Context, _ *v1.Empty) (*v1.GetUsersResponse, error) {
	users, err := s.users.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	resp := &v1.GetUsersResponse{
		Users: users,
	}
	return resp, nil
}

func (s *serviceImpl) GetUser(ctx context.Context, id *v1.ResourceByID) (*storage.User, error) {
	user, err := s.users.GetUser(ctx, id.GetId())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.Wrapf(errox.NotFound, "user %s not found", id.GetId())
	}
	return user, nil
}

func (s *serviceImpl) GetUsersAttributes(ctx context.Context, _ *v1.Empty) (*v1.GetUsersAttributesResponse, error) {
	users, err := s.users.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	attrs := aggregateUserAttributes(users)
	resp := &v1.GetUsersAttributesResponse{
		UsersAttributes: attrs,
	}
	return resp, nil
}

// Helper function that aggregates user attributes.
func aggregateUserAttributes(users []*storage.User) (attrs []*v1.UserAttributeTuple) {
	tups := make(map[string]*v1.UserAttributeTuple)
	for _, user := range users {
		for _, attr := range user.GetAttributes() {
			key := serialize.StringKey(user.GetAuthProviderId(), attr.GetKey(), attr.GetValue())
			if _, exists := tups[key]; !exists {
				tups[key] = &v1.UserAttributeTuple{
					AuthProviderId: user.GetAuthProviderId(),
					Key:            attr.GetKey(),
					Value:          attr.GetValue(),
				}
			}
		}
	}

	attrs = make([]*v1.UserAttributeTuple, 0, len(tups))
	for _, attr := range tups {
		attrs = append(attrs, attr)
	}
	return
}
