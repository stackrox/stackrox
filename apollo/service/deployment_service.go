package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewDeploymentService returns a DeploymentService object.
func NewDeploymentService(storage db.DeploymentStorage) *DeploymentService {
	return &DeploymentService{
		storage: storage,
	}
}

// DeploymentService provides APIs for deployments.
type DeploymentService struct {
	storage db.DeploymentStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *DeploymentService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDeploymentServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *DeploymentService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterDeploymentServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetDeployment returns the deployment with given id.
func (s *DeploymentService) GetDeployment(ctx context.Context, request *v1.GetDeploymentRequest) (*v1.Deployment, error) {
	deployment, exists, err := s.storage.GetDeployment(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "deployment with id '%s' does not exist", request.GetId())
	}

	return deployment, nil
}

// GetDeployments returns deployments according to the request.
func (s *DeploymentService) GetDeployments(ctx context.Context, request *v1.GetDeploymentsRequest) (*v1.GetDeploymentsResponse, error) {
	deployments, err := s.storage.GetDeployments(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1.GetDeploymentsResponse{Deployments: deployments}, nil
}
