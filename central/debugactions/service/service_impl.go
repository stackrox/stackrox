package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/debugactions/manager"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			v2.DebugActionService_GetActionStatus_FullMethodName,
		},
		user.With(permissions.Modify(resources.Administration)): {
			v2.DebugActionService_RegisterAction_FullMethodName,
			v2.DebugActionService_DeleteAction_FullMethodName,
			v2.DebugActionService_ProceedOldest_FullMethodName,
			v2.DebugActionService_ProceedAll_FullMethodName,
		},
	})
)

type serviceImpl struct {
	actionMgr manager.Manager
}

func (s *serviceImpl) RegisterServiceServer(server *grpc.Server) {
	v2.RegisterDebugActionServiceServer(server, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterDebugActionServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) RegisterAction(_ context.Context, action *v2.DebugAction) (*v2.Empty, error) {
	err := s.actionMgr.RegisterAction(action)
	if err != nil {
		return nil, err
	}
	return &v2.Empty{}, nil
}

func (s *serviceImpl) GetActionStatus(_ context.Context, id *v2.ResourceByID) (*v2.ActionStatus, error) {
	return s.actionMgr.GetActionStatus(id.GetId())
}

func (s *serviceImpl) DeleteAction(_ context.Context, id *v2.ResourceByID) (*v2.Empty, error) {
	err := s.actionMgr.DeleteAction(id.GetId())
	if err != nil {
		return nil, err
	}
	return &v2.Empty{}, nil
}

func (s *serviceImpl) ProceedOldest(_ context.Context, id *v2.ResourceByID) (*v2.Empty, error) {
	err := s.actionMgr.ProceedOldest(id.GetId())
	if err != nil {
		return nil, err
	}
	return &v2.Empty{}, nil
}

func (s *serviceImpl) ProceedAll(_ context.Context, id *v2.ResourceByID) (*v2.Empty, error) {
	err := s.actionMgr.ProceedAll(id.GetId())
	if err != nil {
		return nil, err
	}
	return &v2.Empty{}, nil
}
