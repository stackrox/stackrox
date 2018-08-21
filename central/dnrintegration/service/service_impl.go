package service

import (
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/dnrintegration"
	"github.com/stackrox/rox/central/dnrintegration/datastore"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/secrets"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DNRIntegration)): {
			"/v1.DNRIntegrationService/GetDNRIntegration",
			"/v1.DNRIntegrationService/GetDNRIntegrations",
		},
		user.With(permissions.Modify(resources.DNRIntegration)): {
			"/v1.DNRIntegrationService/TestDNRIntegration",
			"/v1.DNRIntegrationService/PostDNRIntegration",
			"/v1.DNRIntegrationService/PutDNRIntegration",
			"/v1.DNRIntegrationService/DeleteDNRIntegration",
		},
	})
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	datastore   datastore.DataStore
	clusters    clusterDataStore.DataStore
	deployments deploymentDataStore.DataStore
	enricher    enrichment.Enricher
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDNRIntegrationServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterDNRIntegrationServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(authorizer.Authorized(ctx, fullMethodName))
}

// GetDNRIntegration retrieves a DNR integration by ID.
func (s *serviceImpl) GetDNRIntegration(ctx context.Context, req *v1.ResourceByID) (*v1.DNRIntegration, error) {
	integration, err := s.getDNRIntegrationByID(req.GetId())
	secrets.ScrubSecretsFromStruct(integration)
	return integration, err
}

// GetDNRIntegrations retrieves all DNR integrations.
func (s *serviceImpl) GetDNRIntegrations(ctx context.Context, req *v1.GetDNRIntegrationsRequest) (*v1.GetDNRIntegrationsResponse, error) {
	integrations, err := s.datastore.GetDNRIntegrations(req)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("couldn't retrieve integrations: %s", err.Error()))
	}
	for _, integration := range integrations {
		secrets.ScrubSecretsFromStruct(integration)
	}
	return &v1.GetDNRIntegrationsResponse{
		Results: integrations,
	}, nil
}

// PostDNRIntegration handles post responses with new DNR integrations.
func (s *serviceImpl) PostDNRIntegration(ctx context.Context, req *v1.DNRIntegration) (*v1.DNRIntegration, error) {
	if req.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("got non-empty id %s in POST", req.GetId()))

	}
	if len(req.GetClusterIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "received empty list of cluster ids")
	}

	err := s.ensureClusterIDsExistAndAreUnique(req.GetClusterIds())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	integration, err := dnrintegration.New(req, s.deployments)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := s.datastore.AddDNRIntegration(req, integration)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	req.Id = id
	go s.enricher.ReprocessRisk()

	return req, nil
}

// PutDNRIntegration updates a DNR integration.
func (s *serviceImpl) PutDNRIntegration(ctx context.Context, req *v1.DNRIntegration) (*v1.DNRIntegration, error) {
	// Make sure the integration exists already.
	_, err := s.getDNRIntegrationByID(req.GetId())
	if err != nil {
		return nil, err
	}

	if len(req.GetClusterIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "received empty list of cluster ids")
	}

	err = s.ensureClusterIDsExistAndHaveOnlyPermittedIntegration(req.GetClusterIds(), req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	integration, err := dnrintegration.New(req, s.deployments)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.datastore.UpdateDNRIntegration(req, integration)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	go s.enricher.ReprocessRisk()

	return req, nil
}

// DeleteDNRIntegration removes a DNR integration by ID.
func (s *serviceImpl) DeleteDNRIntegration(ctx context.Context, req *v1.ResourceByID) (*empty.Empty, error) {
	err := s.datastore.RemoveDNRIntegration(req.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	go s.enricher.ReprocessRisk()

	return &empty.Empty{}, nil
}

// TestDNRIntegration tests the DNR integration.
func (s *serviceImpl) TestDNRIntegration(ctx context.Context, req *v1.DNRIntegration) (*empty.Empty, error) {
	_, err := dnrintegration.New(req, s.deployments)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &empty.Empty{}, nil
}

func (s *serviceImpl) getDNRIntegrationByID(id string) (*v1.DNRIntegration, error) {
	integration, exists, err := s.datastore.GetDNRIntegration(id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("couldn't retrieve integration %s: %s", id, err.Error()))
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("DNR Integration with id %s not found", id))

	}
	return integration, nil
}

func (s *serviceImpl) ensureClusterIDsExistAndHaveOnlyPermittedIntegration(clusterIDs []string, permittedDNRIntegrationID string) error {
	for _, clusterID := range clusterIDs {
		err := s.validateClusterID(clusterID, permittedDNRIntegrationID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *serviceImpl) ensureClusterIDsExistAndAreUnique(clusterIDs []string) error {
	return s.ensureClusterIDsExistAndHaveOnlyPermittedIntegration(clusterIDs, "")
}

// Make sure that the cluster ID exists, and that, if it has a D&R integration, it is only the permitted one.
// Do NOT call this function directly, call it through one of the ensureClusterIDExistsAnd* functions.
func (s *serviceImpl) validateClusterID(clusterID string, permittedDNRIntegrationID string) error {
	_, exists, err := s.clusters.GetCluster(clusterID)
	if err != nil {
		return fmt.Errorf("cluster retrieval: %s", err)
	}
	if !exists {
		return fmt.Errorf("cluster with id %s does not exist", clusterID)
	}
	integrations, err := s.datastore.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{ClusterId: clusterID})
	if err != nil {
		return fmt.Errorf("DNR integration retrieval: %s", err)
	}
	if len(integrations) > 1 {
		return fmt.Errorf("found existing D&R integrations to cluster %s: %v", clusterID, integrations)
	}
	if len(integrations) > 0 {
		if permittedDNRIntegrationID == "" {
			return fmt.Errorf("found existing D&R integration to cluster %s: %v", clusterID, integrations[0])
		}
		if permittedDNRIntegrationID != integrations[0].GetId() {
			return fmt.Errorf("found existing D&R integration to cluster %s: %s", clusterID, integrations[0].GetId())
		}
	}
	return nil
}
