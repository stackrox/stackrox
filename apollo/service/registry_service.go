package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/registries"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewRegistryService returns the RegistryService API.
func NewRegistryService(storage db.Storage) *RegistryService {
	return &RegistryService{
		storage: storage,
	}
}

// RegistryService is the struct that manages the Registry API
type RegistryService struct {
	storage db.RegistryStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *RegistryService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterRegistryServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *RegistryService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterRegistryServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// secretKeys lists the keys that have secret values so we should scrub the keys out of the map before returning from the api
var secretKeys = []string{
	"password",
	"token",
}

// GetRegistries retrieves all registries that matches the request filters
func (s *RegistryService) GetRegistries(ctx context.Context, request *v1.GetRegistriesRequest) (*v1.GetRegistriesResponse, error) {
	registriesWithSecrets := s.storage.GetRegistries()
	registriesWithoutSecrets := make([]*v1.Registry, 0, len(registriesWithSecrets))
	for name, registryWithSecret := range registriesWithSecrets {
		config := registryWithSecret.Config()
		for _, secretKey := range secretKeys {
			delete(config, secretKey)
		}
		registriesWithoutSecrets = append(registriesWithoutSecrets, &v1.Registry{
			Name:     name,
			Endpoint: registryWithSecret.Endpoint(),
			Type:     registryWithSecret.Type(),
			Config:   config,
		})
	}
	return &v1.GetRegistriesResponse{Registries: registriesWithoutSecrets}, nil
}

// PostRegistry inserts a new registry into the system
func (s *RegistryService) PostRegistry(ctx context.Context, request *v1.Registry) (*v1.Registry, error) {
	creator, exists := registries.Registry[request.Type]
	if !exists {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Registry with type %v does not exist", request.Type))
	}
	registry, err := creator(request.Endpoint, request.Config)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.storage.AddRegistry(request.Name, registry)
	return request, nil
}
