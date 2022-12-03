package service

import (
	"context"
	"os"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	datastore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.ProcessListeningOnPort)): {
			"/v1.ProcessesListeningOnPortsService/GetProcessesListeningOnPortsByNamespace",
			"/v1.ProcessesListeningOnPortsService/GetProcessesListeningOnPortsByNamespaceAndDeployment",
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

func IsServiceDisabled() bool {
	if os.Getenv("ROX_POSTGRES_DATASTORE") == "false" {
		log.Warnf("Process listening on port service is disabled when ROX_POSTGRES_DATASTORE is false")
		return true
	}

	if os.Getenv("ROX_PROCESSES_LISTENING_ON_PORT") == "false" {
		log.Warnf("Process listening on port service is disabled when ROX_PROCESSES_LISTENING_ON_PORT is false")
		return true
	}

	return false
}

func EmptyProcessesListeningOnPortsWithDeploymentResponse() (*v1.GetProcessesListeningOnPortsWithDeploymentResponse, error) {
	result := &v1.GetProcessesListeningOnPortsWithDeploymentResponse{
		ProcessesListeningOnPortsWithDeployment: make([]*v1.ProcessListeningOnPortWithDeploymentId, 0),
	}
	return result, nil
}


func (s *serviceImpl) GetProcessesListeningOnPortsByNamespace(ctx context.Context, req *v1.GetProcessesListeningOnPortsByNamespaceRequest) (*v1.GetProcessesListeningOnPortsWithDeploymentResponse, error) {
	namespace := req.GetNamespace()

	if IsServiceDisabled() {
		return EmptyProcessesListeningOnPortsWithDeploymentResponse()
	}

	processesListeningOnPorts, err := s.dataStore.GetProcessListeningOnPort(
		ctx, datastore.GetOptions{Namespace: &namespace})

	if err != nil {
		log.Warnf("In processlisteningonport service query return err: %+v", err)
		return EmptyProcessesListeningOnPortsWithDeploymentResponse()
	}

	if processesListeningOnPorts == nil {
		log.Debug("In processlisteningonport service query return nil")
		return EmptyProcessesListeningOnPortsWithDeploymentResponse()
	}

	result := make([]*v1.ProcessListeningOnPortWithDeploymentId, 0)

	for k, v := range processesListeningOnPorts {
		plop := &v1.ProcessListeningOnPortWithDeploymentId{
			DeploymentId:              k,
			ProcessesListeningOnPorts: v,
		}
		result = append(result, plop)
	}

	return &v1.GetProcessesListeningOnPortsWithDeploymentResponse{
		ProcessesListeningOnPortsWithDeployment: result,
	}, err
}

func EmptyProcessesListeningOnPortsResponse() (*v1.GetProcessesListeningOnPortsResponse, error) {
	result := &v1.GetProcessesListeningOnPortsResponse{
		ProcessesListeningOnPorts: make([]*storage.ProcessListeningOnPort, 0),
	}
	return result, nil
}

func (s *serviceImpl) GetProcessesListeningOnPortsByNamespaceAndDeployment(
	ctx context.Context,
	req *v1.GetProcessesListeningOnPortsByNamespaceAndDeploymentRequest,
) (*v1.GetProcessesListeningOnPortsResponse, error) {

	if IsServiceDisabled() {
		log.Warnf("Process listening on port service is disabled when ROX_POSTGRES_DATASTORE or ROX_PROCESSES_LISTENING_ON_PORT is false")
		return EmptyProcessesListeningOnPortsResponse()
	}

	namespace := req.GetNamespace()
	deployment := req.GetDeploymentId()
	processesListeningOnPorts, err := s.dataStore.GetProcessListeningOnPort(
		ctx, datastore.GetOptions{
			Namespace:    &namespace,
			DeploymentID: &deployment,
		})
	log.Info("In processlisteningonport service got processes")

	if err != nil {
		log.Warnf("In processlisteningonport service query return err: %+v", err)
		return EmptyProcessesListeningOnPortsResponse()
	}

	if processesListeningOnPorts == nil {
		log.Debug("In processlisteningonport service query return nil")
		return EmptyProcessesListeningOnPortsResponse()
	}

	result := make([]*storage.ProcessListeningOnPort, 0)

	// Storage returns map DeploymentID -> PLOP. Just in case verify that
	// deployment id matches.
	for k, v := range processesListeningOnPorts {
		if k != deployment {
			log.Warnf("Requested deployment %s, got %s. Skipping", deployment, k)
		} else {
			result = append(result, v...)
		}
	}

	return &v1.GetProcessesListeningOnPortsResponse{
		ProcessesListeningOnPorts: result,
	}, err
}
