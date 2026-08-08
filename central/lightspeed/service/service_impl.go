package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/lightspeed/store"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.Modify(resources.Administration)): {
			v1.LightspeedService_ConfigureLightspeed_FullMethodName,
		},
		user.With(permissions.View(resources.Administration)): {
			v1.LightspeedService_GetLightspeedConfig_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedLightspeedServiceServer

	store   *store.Store
	connMgr connection.Manager
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

func (s *serviceImpl) ConfigureLightspeed(ctx context.Context, req *v1.ConfigureLightspeedRequest) (*v1.Empty, error) {
	clusterID := req.GetClusterId()
	host := req.GetHost()
	if clusterID == "" {
		return nil, errox.InvalidArgs.New("cluster_id is required")
	}

	s.store.SetHost(clusterID, host)
	log.Infof("Lightspeed config updated for cluster %s: host=%s", clusterID, host)

	conn := s.connMgr.GetConnection(clusterID)
	if conn == nil {
		return &v1.Empty{}, nil
	}

	err := conn.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_LightspeedConfig{
			LightspeedConfig: &central.LightspeedConfig{
				Host: host,
			},
		},
	})
	if err != nil {
		log.Warnf("Failed to send Lightspeed config to sensor for cluster %s: %v", clusterID, err)
	}

	return &v1.Empty{}, nil
}

func (s *serviceImpl) GetLightspeedConfig(_ context.Context, req *v1.ResourceByID) (*v1.GetLightspeedConfigResponse, error) {
	clusterID := req.GetId()
	if clusterID == "" {
		return nil, errox.InvalidArgs.New("cluster id is required")
	}

	host, info := s.store.Get(clusterID)
	resp := &v1.GetLightspeedConfigResponse{
		Host: host,
	}
	if info != nil {
		resp.IsReady = info.GetIsReady()
		resp.HasQueryAccess = info.GetHasQueryAccess()
		resp.StatusError = info.GetStatusError()
	}
	return resp, nil
}
