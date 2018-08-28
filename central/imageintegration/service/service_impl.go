package service

import (
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichanddetect"
	"github.com/stackrox/rox/central/imageintegration/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/secrets"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		or.SensorOrAuthorizer(user.With(permissions.View(resources.ImageIntegration))): {
			"/v1.ImageIntegrationService/GetImageIntegration",
			"/v1.ImageIntegrationService/GetImageIntegrations",
		},
		user.With(permissions.Modify(resources.ImageIntegration)): {
			"/v1.ImageIntegrationService/PostImageIntegration",
			"/v1.ImageIntegrationService/PutImageIntegration",
			"/v1.ImageIntegrationService/TestImageIntegration",
			"/v1.ImageIntegrationService/DeleteImageIntegration",
		},
	})
)

// ImageIntegrationService is the struct that manages the ImageIntegration API
type serviceImpl struct {
	registryFactory registries.Factory
	scannerFactory  scanners.Factory
	toNotify        integration.ToNotify

	datastore           datastore.DataStore
	clusterDatastore    clusterDatastore.DataStore
	enrichAndDetectLoop enrichanddetect.Loop
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterImageIntegrationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterImageIntegrationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func scrubImageIntegration(i *v1.ImageIntegration) {
	i.Config = secrets.ScrubSecretsFromMap(i.Config)
	secrets.ScrubSecretsFromStruct(i)
}

// GetImageIntegration retrieves the integration based on the id passed
func (s *serviceImpl) GetImageIntegration(ctx context.Context, request *v1.ResourceByID) (*v1.ImageIntegration, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Image integration id must be provided")
	}
	integration, exists, err := s.datastore.GetImageIntegration(request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Image integration %s not found", request.GetId()))
	}
	scrubImageIntegration(integration)
	return integration, nil
}

// GetImageIntegrations retrieves all image integrations that matches the request filters
func (s *serviceImpl) GetImageIntegrations(ctx context.Context, request *v1.GetImageIntegrationsRequest) (*v1.GetImageIntegrationsResponse, error) {
	integrations, err := s.datastore.GetImageIntegrations(request)
	if err != nil {
		return nil, err
	}

	identity, err := authn.FromTLSContext(ctx)
	switch {
	case err == authn.ErrNoContext:
		log.Debugf("No authentication context provided")
	case err != nil:
		log.Warnf("Error getting client identity: %s", err)
	case err == nil && identity.Name.ServiceType == v1.ServiceType_SENSOR_SERVICE:
		return &v1.GetImageIntegrationsResponse{Integrations: integrations}, nil
	}

	// Remove secrets for other API accessors.
	for _, i := range integrations {
		scrubImageIntegration(i)
	}
	return &v1.GetImageIntegrationsResponse{Integrations: integrations}, nil
}

// PutImageIntegration updates an image integration in the system
func (s *serviceImpl) PutImageIntegration(ctx context.Context, request *v1.ImageIntegration) (*v1.Empty, error) {
	err := s.validateClustersAndCategories(request)
	if err != nil {
		return nil, err
	}

	if err := s.datastore.UpdateImageIntegration(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := s.toNotify.NotifyUpdated(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	go s.enrichAndDetectLoop.ShortCircuit()
	return &v1.Empty{}, nil
}

// PostImageIntegration inserts a new image integration into the system if it doesn't already exist
func (s *serviceImpl) PostImageIntegration(ctx context.Context, request *v1.ImageIntegration) (*v1.ImageIntegration, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new image integration")
	}

	err := s.validateClustersAndCategories(request)
	if err != nil {
		return nil, err
	}

	id, err := s.datastore.AddImageIntegration(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	request.Id = id

	if err := s.toNotify.NotifyUpdated(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	go s.enrichAndDetectLoop.ShortCircuit()
	return request, nil
}

// DeleteImageIntegration deletes an integration from the system
func (s *serviceImpl) DeleteImageIntegration(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Image integration id must be provided")
	}
	if err := s.datastore.RemoveImageIntegration(request.GetId()); err != nil {
		return nil, err
	}
	if err := s.toNotify.NotifyRemoved(request.GetId()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.Empty{}, nil
}

// TestImageIntegration tests to see if the config is setup properly
func (s *serviceImpl) TestImageIntegration(ctx context.Context, request *v1.ImageIntegration) (*v1.Empty, error) {
	err := s.validateClustersAndCategories(request)
	if err != nil {
		return nil, err
	}
	for _, category := range request.GetCategories() {
		if category == v1.ImageIntegrationCategory_REGISTRY {
			err = s.testRegistryIntegration(request)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		if category == v1.ImageIntegrationCategory_SCANNER {
			err = s.testScannerIntegration(request)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) testRegistryIntegration(integration *v1.ImageIntegration) error {
	registry, err := s.registryFactory.CreateRegistry(integration)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if err := registry.Test(); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

func (s *serviceImpl) testScannerIntegration(integration *v1.ImageIntegration) error {
	scanner, err := s.scannerFactory.CreateScanner(integration)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if err := scanner.Test(); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

func (s *serviceImpl) validateClustersAndCategories(request *v1.ImageIntegration) error {
	if len(request.GetCategories()) == 0 {
		return status.Error(codes.InvalidArgument, "integrations require a category")
	}

	clustersRequested := request.GetClusters()
	existingClusters, err := s.clusterDatastore.GetClusters()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	for _, req := range clustersRequested {
		if !s.clusterExists(req, existingClusters) {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("Cluster %s does not exist", req))
		}
	}

	return nil
}

func (s *serviceImpl) clusterExists(name string, clusters []*v1.Cluster) bool {
	for _, c := range clusters {
		if name == c.GetName() {
			return true
		}
	}
	return false
}
