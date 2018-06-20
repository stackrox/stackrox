package service

import (
	"context"
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/dnr_integration"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewDNRIntegrationService returns a DNR integration service object.
func NewDNRIntegrationService(storage db.DNRIntegrationStorage, clusterStorage db.ClusterStorage) *DNRIntegrationService {
	return &DNRIntegrationService{
		storage:        storage,
		clusterStorage: clusterStorage,
	}
}

// DNRIntegrationService helps integrate with Detect & Respond
type DNRIntegrationService struct {
	storage        db.DNRIntegrationStorage
	clusterStorage db.ClusterStorage
}

// GetDNRIntegration retrieves a DNR integration by ID.
func (s *DNRIntegrationService) GetDNRIntegration(ctx context.Context, req *v1.ResourceByID) (*v1.DNRIntegration, error) {
	return s.getDNRIntegrationByID(req.GetId())
}

// GetDNRIntegrations retrieves all DNR integrations.
func (s *DNRIntegrationService) GetDNRIntegrations(ctx context.Context, req *v1.GetDNRIntegrationsRequest) (*v1.GetDNRIntegrationsResponse, error) {
	integrations, err := s.storage.GetDNRIntegrations(req)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("couldn't retrieve integrations: %s", err.Error()))
	}
	return &v1.GetDNRIntegrationsResponse{
		Results: integrations,
	}, nil
}

// PostDNRIntegration handles post responses with new DNR integrations.
func (s *DNRIntegrationService) PostDNRIntegration(ctx context.Context, req *v1.DNRIntegration) (*v1.DNRIntegration, error) {
	if req.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("got non-empty id %s in POST", req.GetId()))

	}
	err := s.ensureClusterIDExistsAndIsUnique(req.GetClusterId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	id, err := s.storage.AddDNRIntegration(req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	req.Id = id
	return req, nil
}

// PutDNRIntegration updates a DNR integration.
func (s *DNRIntegrationService) PutDNRIntegration(ctx context.Context, req *v1.DNRIntegration) (*v1.DNRIntegration, error) {
	// Make sure the integration exists already.
	_, err := s.getDNRIntegrationByID(req.GetId())
	if err != nil {
		return nil, err
	}

	err = s.ensureClusterIDExistsAndHasOnlyPermittedIntegration(req.GetClusterId(), req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.storage.UpdateDNRIntegration(req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return req, nil
}

// DeleteDNRIntegration removes a DNR integration by ID.
func (s *DNRIntegrationService) DeleteDNRIntegration(ctx context.Context, req *v1.ResourceByID) (*empty.Empty, error) {
	err := s.storage.RemoveDNRIntegration(req.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &empty.Empty{}, nil
}

// TestDNRIntegration tests the DNR integration.
func (s *DNRIntegrationService) TestDNRIntegration(ctx context.Context, req *v1.DNRIntegration) (*empty.Empty, error) {
	integration, err := dnrintegration.New(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	err = integration.Test()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &empty.Empty{}, nil
}

func (s *DNRIntegrationService) getDNRIntegrationByID(id string) (*v1.DNRIntegration, error) {
	integration, exists, err := s.storage.GetDNRIntegration(id)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("couldn't retrieve integration %s: %s", id, err.Error()))
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("DNR Integration with id %s not found", id))

	}
	return integration, nil
}

func (s *DNRIntegrationService) ensureClusterIDExistsAndHasOnlyPermittedIntegration(clusterID, permittedDNRIntegrationID string) error {
	return s.validateClusterID(clusterID, permittedDNRIntegrationID)
}

func (s *DNRIntegrationService) ensureClusterIDExistsAndIsUnique(clusterID string) error {
	return s.validateClusterID(clusterID, "")
}

// Make sure that the cluster ID exists, and that, if it has a D&R integration, it is only the permitted one.
// Do NOT call this function directly, call it through one of the ensureClusterIDExistsAnd* functions.
func (s *DNRIntegrationService) validateClusterID(clusterID string, permittedDNRIntegrationID string) error {
	_, exists, err := s.clusterStorage.GetCluster(clusterID)
	if err != nil {
		return fmt.Errorf("cluster retrieval: %s", err)
	}
	if !exists {
		return fmt.Errorf("cluster with id %s does not exist", clusterID)
	}
	integrations, err := s.storage.GetDNRIntegrations(&v1.GetDNRIntegrationsRequest{ClusterId: clusterID})
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

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *DNRIntegrationService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDNRIntegrationServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *DNRIntegrationService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterDNRIntegrationServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *DNRIntegrationService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, ReturnErrorCode(user.Any().Authorized(ctx))
}
