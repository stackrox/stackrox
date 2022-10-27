package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	 datastore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"google.golang.org/grpc"
)

type serviceImpl struct{
	dataStore         datastore.DataStore
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
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetProcessesListeningOnPortsByNamespace(ctx context.Context, req *v1.GetProcessesListeningOnPortsByNamespaceRequest) (*v1.GetProcessesListeningOnPortsWithDeploymentResponse, error) {
	//processIndicatorUniqueKey := &storage.ProcessIndicatorUniqueKey{
	//	PodId:               "nginx-5da7f5-fdsaf",
	//	ContainerName:       "nginx",
	//	ProcessName:         "nginx",
	//	ProcessExecFilePath: "/usr/bin/nginx",
	//	ProcessArgs:         "fake args",
	//}

	//processListeningOnPort := &storage.ProcessListeningOnPort{
	//	Port:           80,
	//	Protocol:       storage.L4Protocol_L4_PROTOCOL_UDP,
	//	Process:        processIndicatorUniqueKey,
	//	CloseTimestamp: nil,
	//}

	//processListeningOnPortWithDeploymentID := &v1.ProcessListeningOnPortWithDeploymentId{
	//	DeploymentId:              "nginx",
	//	ProcessesListeningOnPorts: []*storage.ProcessListeningOnPort{processListeningOnPort},
	//}

	//result := &v1.GetProcessesListeningOnPortsWithDeploymentResponse{
	//	ProcessesListeningOnPortsWithDeployment: []*v1.ProcessListeningOnPortWithDeploymentId{processListeningOnPortWithDeploymentID},
	//}

	log.Info("In processlisteningonport service about to get processes namespace level")
	namespace := req.GetNamespace()
	processesListeningOnPorts, err := s.dataStore.GetProcessListeningOnPortForNamespace(ctx, namespace);
	log.Info("In processlisteningonport service got processes namespace level")

	if err != nil {
		log.Info("In processlisteningonport service query return err")
		log.Info("%v", err)
		result := &v1.GetProcessesListeningOnPortsWithDeploymentResponse{
			ProcessesListeningOnPortsWithDeployment: make([]*v1.ProcessListeningOnPortWithDeploymentId, 0),
		}
		return result, nil
	}

	if processesListeningOnPorts == nil {
		log.Info("In processlisteningonport service query return nil")
		result := &v1.GetProcessesListeningOnPortsWithDeploymentResponse{
			ProcessesListeningOnPortsWithDeployment: make([]*v1.ProcessListeningOnPortWithDeploymentId, 0),
		}
		return result, nil
	}

	result := &v1.GetProcessesListeningOnPortsWithDeploymentResponse{
		ProcessesListeningOnPortsWithDeployment: processesListeningOnPorts,
	}

	return result, err
}

func (s *serviceImpl) GetProcessesListeningOnPortsByNamespaceAndDeployment(ctx context.Context, req *v1.GetProcessesListeningOnPortsByNamespaceAndDeploymentRequest) (*v1.GetProcessesListeningOnPortsResponse, error) {

	log.Info("In processlisteningonport service about to get processes")
	namespace := req.GetNamespace()
	deployment := req.GetDeploymentId()
	processesListeningOnPorts, err := s.dataStore.GetProcessListeningOnPortForDeployment(ctx, namespace, deployment);
	log.Info("In processlisteningonport service got processes")

	if err != nil {
		log.Info("In processlisteningonport service query return err")
		log.Info("%v", err)
		result := &v1.GetProcessesListeningOnPortsResponse{
			ProcessesListeningOnPorts: make([]*storage.ProcessListeningOnPort, 0),
		}
		return result, nil
	}

	if processesListeningOnPorts == nil {
		log.Info("In processlisteningonport service query return nil")
		result := &v1.GetProcessesListeningOnPortsResponse{
			ProcessesListeningOnPorts: make([]*storage.ProcessListeningOnPort, 0),
		}
		return result, nil
	}

	result := &v1.GetProcessesListeningOnPortsResponse{
		ProcessesListeningOnPorts: processesListeningOnPorts,
	}

	return result, err
}
