package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/lightspeed/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
	user.With(permissions.View(resources.Integration)): {
		v1.LightspeedService_GetLightspeedStatus_FullMethodName,
	},
})

type serviceImpl struct {
	v1.UnimplementedLightspeedServiceServer

	datastore datastore.DataStore
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterLightspeedServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterLightspeedServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetLightspeedStatus(_ context.Context, _ *v1.Empty) (*v1.LightspeedStatusResponse, error) {
	for _, info := range s.datastore.GetAll() {
		if info.GetIsAvailable() {
			return &v1.LightspeedStatusResponse{
				Available: true,
				Endpoint:  info.GetEndpoint(),
			}, nil
		}
	}
	return &v1.LightspeedStatusResponse{
		Available: false,
	}, nil
}
