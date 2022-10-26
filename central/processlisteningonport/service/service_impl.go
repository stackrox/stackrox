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

func (s *serviceImpl) GetProcessesListeningOnPortsByNamespace(context.Context, *v1.GetProcessesListeningOnPortsByNamespaceRequest) (*v1.GetProcessesListeningOnPortsWithDeploymentResponse, error) {
	processIndicatorUniqueKey := &storage.ProcessIndicatorUniqueKey{
		PodId:               "nginx-5da7f5-fdsaf",
		ContainerName:       "nginx",
		ProcessName:         "nginx",
		ProcessExecFilePath: "/usr/bin/nginx",
		ProcessArgs:         "fake args",
	}

	processListeningOnPort := &storage.ProcessListeningOnPort{
		Port:           80,
		Protocol:       storage.L4Protocol_L4_PROTOCOL_UDP,
		Process:        processIndicatorUniqueKey,
		CloseTimestamp: nil,
	}

	processListeningOnPortWithDeploymentID := &v1.ProcessListeningOnPortWithDeploymentId{
		DeploymentId:              "nginx",
		ProcessesListeningOnPorts: []*storage.ProcessListeningOnPort{processListeningOnPort},
	}

	result := &v1.GetProcessesListeningOnPortsWithDeploymentResponse{
		ProcessesListeningOnPortsWithDeployment: []*v1.ProcessListeningOnPortWithDeploymentId{processListeningOnPortWithDeploymentID},
	}
	return result, nil
}

func (s *serviceImpl) GetProcessesListeningOnPortsByNamespaceAndDeployment(ctx context.Context, req *v1.GetProcessesListeningOnPortsByNamespaceAndDeploymentRequest) (*v1.GetProcessesListeningOnPortsResponse, error) {
	//processIndicatorUniqueKey1 := &storage.ProcessIndicatorUniqueKey{
	//	PodId:               "nginx-5da7f5-fdsaf",
	//	ContainerName:       "nginx",
	//	ProcessName:         "nginx",
	//	ProcessExecFilePath: "/usr/bin/nginx",
	//	ProcessArgs:         "fake args",
	//}

	//processListeningOnPort1 := &storage.ProcessListeningOnPort{
	//	Port:           80,
	//	Protocol:       storage.L4Protocol_L4_PROTOCOL_UDP,
	//	Process:        processIndicatorUniqueKey1,
	//	CloseTimestamp: nil,
	//}

	//processIndicatorUniqueKey2 := &storage.ProcessIndicatorUniqueKey{
	//	PodId:               "visa-5da7f5-fdsaf",
	//	ContainerName:       "visa",
	//	ProcessName:         "visa",
	//	ProcessExecFilePath: "/usr/bin/visa",
	//	ProcessArgs:         "fake args for visa",
	//}

	//processListeningOnPort2 := &storage.ProcessListeningOnPort{
	//	Port:           8080,
	//	Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
	//	Process:        processIndicatorUniqueKey2,
	//	CloseTimestamp: nil,
	//}

	//result := &v1.GetProcessesListeningOnPortsResponse{
	//	ProcessesListeningOnPorts: []*storage.ProcessListeningOnPort{processListeningOnPort1, processListeningOnPort2},
	//}

	processesListeningOnPorts, err := s.dataStore.GetProcessListeningOnPortForDeployment(ctx, "nginx");

	result := &v1.GetProcessesListeningOnPortsResponse{
		ProcessesListeningOnPorts: []*storage.ProcessListeningOnPort{processesListeningOnPorts},
	}

	return result, err
}
