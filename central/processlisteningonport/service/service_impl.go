package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	datastore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DeploymentExtension)): {
			"/v1.ProcessesListeningOnPortsService/GetProcessesListeningOnPorts",
		},
	})
)

type serviceImpl struct {
	dataStore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterProcessesListeningOnPortsServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterProcessesListeningOnPortsServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func emptyProcessesListeningOnPortsResponse() (*v1.GetProcessesListeningOnPortsResponse, error) {
	result := &v1.GetProcessesListeningOnPortsResponse{
		ListeningEndpoints: make([]*storage.ProcessListeningOnPort, 0),
	}
	return result, nil
}

func (s *serviceImpl) GetProcessesListeningOnPorts(
	ctx context.Context,
	req *v1.GetProcessesListeningOnPortsRequest,
) (*v1.GetProcessesListeningOnPortsResponse, error) {

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		// PLOP is a Postgres-only feature, do nothing.
		log.Warnf("Tried to request PLOP not on Postgres, ignore: %s", req.GetDeploymentId())
		return emptyProcessesListeningOnPortsResponse()
	}

	deployment := req.GetDeploymentId()
	processesListeningOnPorts, err := s.dataStore.GetProcessListeningOnPort(ctx, deployment)

	if err != nil {
		log.Warnf("In processlisteningonport service query return err: %+v", err)
		return emptyProcessesListeningOnPortsResponse()
	}

	if processesListeningOnPorts == nil {
		log.Debug("In processlisteningonport service query return nil")
		return emptyProcessesListeningOnPortsResponse()
	}

	return &v1.GetProcessesListeningOnPortsResponse{
		ListeningEndpoints: processesListeningOnPorts,
	}, err
}
