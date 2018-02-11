package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/detection"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authn"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/or"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/secrets"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewRegistryService returns the RegistryService API.
func NewRegistryService(storage db.RegistryStorage, detection *detection.Detector) *RegistryService {
	return &RegistryService{
		storage:  storage,
		detector: detection,
	}
}

// RegistryService is the struct that manages the Registry API
type RegistryService struct {
	storage  db.RegistryStorage
	detector *detection.Detector
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *RegistryService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterRegistryServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *RegistryService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterRegistryServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *RegistryService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(or.SensorOrUser().Authorized(ctx))
}

// GetRegistry retrieves the registry based on the id passed
func (s *RegistryService) GetRegistry(ctx context.Context, request *v1.ResourceByID) (*v1.Registry, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Registry id must be provided")
	}
	registry, exists, err := s.storage.GetRegistry(request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Scanner %v not found", request.GetId()))
	}
	return registry, nil
}

// GetRegistries retrieves all registries that matches the request filters
func (s *RegistryService) GetRegistries(ctx context.Context, request *v1.GetRegistriesRequest) (*v1.GetRegistriesResponse, error) {
	registries, err := s.storage.GetRegistries(request)
	if err != nil {
		return nil, err
	}

	identity, err := authn.FromTLSContext(ctx)
	switch {
	case err == authn.ErrNoContext:
		log.Debugf("No authentication context provided")
	case err != nil:
		log.Warnf("Could not ascertain client identity: %s", err)
	case err == nil && identity.Name.ServiceType == v1.ServiceType_SENSOR_SERVICE:
		return &v1.GetRegistriesResponse{Registries: registries}, nil
	}

	// Remove secrets for other API accessors.
	for _, r := range registries {
		r.Config = secrets.ScrubSecrets(r.Config)
	}
	return &v1.GetRegistriesResponse{Registries: registries}, nil
}

// PutRegistry updates a registry in the system
func (s *RegistryService) PutRegistry(ctx context.Context, request *v1.Registry) (*empty.Empty, error) {
	// creates and validates the configuration
	registry, err := registries.CreateRegistry(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.storage.UpdateRegistry(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.detector.UpdateRegistry(registry)
	return &empty.Empty{}, nil
}

// PostRegistry inserts a new registry into the system if it doesn't already exist
func (s *RegistryService) PostRegistry(ctx context.Context, request *v1.Registry) (*v1.Registry, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new registry")
	}
	// creates and validates the configuration
	registry, err := registries.CreateRegistry(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	id, err := s.storage.AddRegistry(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	request.Id = id
	s.detector.UpdateRegistry(registry)
	return request, nil
}

// DeleteRegistry deletes a registry from the system
func (s *RegistryService) DeleteRegistry(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Registry id must be provided")
	}
	if err := s.storage.RemoveRegistry(request.GetId()); err != nil {
		return nil, returnErrorCode(err)
	}
	s.detector.RemoveRegistry(request.GetId())
	return &empty.Empty{}, nil
}
