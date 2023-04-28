package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	datastore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
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
			"/v1.ListeningEndpointsService/GetListeningEndpoints",
		},
	})
)

type serviceImpl struct {
	dataStore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterListeningEndpointsServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterListeningEndpointsServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetListeningEndpoints returns the listening endpoints and the processes that opened them for a given deployment
func (s *serviceImpl) GetListeningEndpoints(
	ctx context.Context,
	req *v1.GetProcessesListeningOnPortsRequest,
) (*v1.GetProcessesListeningOnPortsResponse, error) {

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		// PLOP is a Postgres-only feature, do nothing.
		return nil, errors.New("Postgres env var is not enabled, PLOP APIs are disabled")
	}

	deployment := req.GetDeploymentId()
	processesListeningOnPorts, err := s.dataStore.GetProcessListeningOnPort(ctx, deployment)

	if err != nil {
		return nil, err
	}

	return &v1.GetProcessesListeningOnPortsResponse{
		ListeningEndpoints: processesListeningOnPorts,
	}, nil
}
