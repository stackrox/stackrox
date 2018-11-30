package service

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/group/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
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
		user.With(permissions.View(resources.Group)): {
			"/v1.GroupService/GetGroups",
			"/v1.GroupService/GetGroup",
		},
		user.With(permissions.Modify(resources.Group)): {
			"/v1.GroupService/CreateGroup",
			"/v1.GroupService/UpdateGroup",
			"/v1.GroupService/DeleteGroup",
		},
	})
)

type serviceImpl struct {
	groupStore store.Store
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterGroupServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterGroupServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetGroups(context.Context, *v1.Empty) (*v1.GetGroupsResponse, error) {
	groups, err := s.groupStore.GetAll()
	if err != nil {
		return nil, err
	}
	return &v1.GetGroupsResponse{Groups: groups}, nil
}

func (s *serviceImpl) GetGroup(ctx context.Context, props *v1.GroupProperties) (*v1.Group, error) {
	group, err := s.groupStore.Get(props)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, status.Errorf(codes.NotFound, "group %q not found", proto.MarshalTextString(props))
	}
	return group, nil
}

func (s *serviceImpl) CreateGroup(ctx context.Context, group *v1.Group) (*v1.Empty, error) {
	if err := validate(group); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	err := s.groupStore.Add(group)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) UpdateGroup(ctx context.Context, group *v1.Group) (*v1.Empty, error) {
	if err := validate(group); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	err := s.groupStore.Update(group)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteGroup(ctx context.Context, props *v1.GroupProperties) (*v1.Empty, error) {
	err := s.groupStore.Remove(props)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}
