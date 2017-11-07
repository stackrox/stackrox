package service

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/image_processor"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewRegistryService returns the RegistryService API.
func NewRegistryService(storage db.Storage, processor *imageprocessor.ImageProcessor) *RegistryService {
	return &RegistryService{
		storage:   storage,
		processor: processor,
	}
}

// RegistryService is the struct that manages the Registry API
type RegistryService struct {
	storage   db.Storage
	processor *imageprocessor.ImageProcessor
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *RegistryService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterRegistryServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *RegistryService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterRegistryServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetRegistries retrieves all registries that matches the request filters
func (s *RegistryService) GetRegistries(ctx context.Context, request *v1.GetRegistriesRequest) (*v1.GetRegistriesResponse, error) {
	return &v1.GetRegistriesResponse{}, status.Error(codes.Unimplemented, "Not implemented")
}

// PostRegistry inserts a new registry into the system
func (s *RegistryService) PostRegistry(ctx context.Context, request *v1.Registry) (*v1.Registry, error) {
	return request, status.Error(codes.Unimplemented, "Not implemented")
}
