package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	datastore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DeploymentExtension)): {
			v1.ListeningEndpointsService_GetListeningEndpoints_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedListeningEndpointsServiceServer
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
	deployment := req.GetDeploymentId()
	page := req.GetPagination()
	processesListeningOnPorts, err := s.dataStore.GetProcessListeningOnPort(ctx, deployment)
	totalListeningEndpoints := len(processesListeningOnPorts)

	if err != nil {
		return nil, err
	}

	if page != nil {
		processesListeningOnPorts = paginated.PaginateSlice(int(page.GetOffset()), int(page.GetLimit()), processesListeningOnPorts)
	}

	return &v1.GetProcessesListeningOnPortsResponse{
		ListeningEndpoints:      processesListeningOnPorts,
		TotalListeningEndpoints: int32(totalListeningEndpoints),
	}, nil
}
