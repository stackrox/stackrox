package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/aiintegration/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/tlsutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	testTimeout = 10 * time.Second
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Integration)): {
			v2.AiIntegrationService_GetAiIntegration_FullMethodName,
			v2.AiIntegrationService_ListAiIntegrations_FullMethodName,
		},
		user.With(permissions.Modify(resources.Integration)): {
			v2.AiIntegrationService_CreateAiIntegration_FullMethodName,
			v2.AiIntegrationService_UpdateAiIntegration_FullMethodName,
			v2.AiIntegrationService_DeleteAiIntegration_FullMethodName,
			v2.AiIntegrationService_TestAiIntegration_FullMethodName,
		},
	})
)

// New returns a new AI integration service instance.
func New(ds datastore.DataStore) Service {
	return &serviceImpl{
		datastore: ds,
	}
}

type serviceImpl struct {
	v2.UnimplementedAiIntegrationServiceServer

	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterAiIntegrationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterAiIntegrationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// CreateAiIntegration creates a new AI integration.
// Only one AI integration is allowed at a time.
func (s *serviceImpl) CreateAiIntegration(ctx context.Context, req *v2.AiIntegration) (*v2.AiIntegration, error) {
	if err := validateIntegration(req); err != nil {
		return nil, err
	}

	existing, err := s.datastore.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, errors.Wrap(errox.AlreadyExists, "only one AI integration is allowed; delete the existing integration before creating a new one")
	}

	storageObj := apiToStorage(req)
	id, err := s.datastore.Add(ctx, storageObj)
	if err != nil {
		return nil, err
	}

	log.Infof("Created AI integration %q (id=%s, type=%s)", storageObj.GetName(), id, storageObj.GetType())
	return storageToAPI(storageObj), nil
}

// ListAiIntegrations returns all configured AI integrations.
func (s *serviceImpl) ListAiIntegrations(ctx context.Context, _ *v2.Empty) (*v2.ListAiIntegrationsResponse, error) {
	integrations, err := s.datastore.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*v2.AiIntegration, 0, len(integrations))
	for _, integration := range integrations {
		result = append(result, storageToAPI(integration))
	}

	return &v2.ListAiIntegrationsResponse{Integrations: result}, nil
}

// GetAiIntegration returns a single AI integration by ID.
func (s *serviceImpl) GetAiIntegration(ctx context.Context, req *v2.ResourceByID) (*v2.AiIntegration, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "integration id must be provided")
	}

	integration, exists, err := s.datastore.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "integration %q not found", req.GetId())
	}

	return storageToAPI(integration), nil
}

// UpdateAiIntegration updates an existing AI integration.
func (s *serviceImpl) UpdateAiIntegration(ctx context.Context, req *v2.AiIntegration) (*v2.Empty, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "integration id must be provided")
	}
	if err := validateIntegration(req); err != nil {
		return nil, err
	}

	exists, err := s.datastore.Exists(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "integration %q not found", req.GetId())
	}

	if err := s.datastore.Upsert(ctx, apiToStorage(req)); err != nil {
		return nil, err
	}

	log.Infof("Updated AI integration %q (id=%s)", req.GetName(), req.GetId())
	return &v2.Empty{}, nil
}

// DeleteAiIntegration deletes an AI integration by ID.
func (s *serviceImpl) DeleteAiIntegration(ctx context.Context, req *v2.ResourceByID) (*v2.Empty, error) {
	if req.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "integration id must be provided")
	}

	exists, err := s.datastore.Exists(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "integration %q not found", req.GetId())
	}

	if err := s.datastore.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}

	log.Infof("Deleted AI integration (id=%s)", req.GetId())
	return &v2.Empty{}, nil
}

// TestAiIntegration tests connectivity to an AI integration endpoint.
func (s *serviceImpl) TestAiIntegration(_ context.Context, req *v2.AiIntegration) (*v2.Empty, error) {

	if err := validateIntegration(req); err != nil {
		return nil, err
	}

	if err := testConnectivity(req.GetServiceUrl()); err != nil {
		return nil, status.Errorf(codes.Unavailable, "connectivity test failed: %v", err)
	}

	return &v2.Empty{}, nil
}

func validateIntegration(integration *v2.AiIntegration) error {
	if integration == nil {
		return errors.Wrap(errox.InvalidArgs, "integration must be provided")
	}
	if integration.GetName() == "" {
		return errors.Wrap(errox.InvalidArgs, "integration name must be provided")
	}
	if integration.GetType() == v2.AiIntegrationType_AI_INTEGRATION_TYPE_UNSPECIFIED {
		return errors.Wrap(errox.InvalidArgs, "integration type must be specified")
	}
	if integration.GetServiceUrl() == "" {
		return errors.Wrap(errox.InvalidArgs, "service_url must be provided")
	}
	if _, err := url.ParseRequestURI(integration.GetServiceUrl()); err != nil {
		return errors.Wrap(errox.InvalidArgs, fmt.Sprintf("invalid service_url: %v", err))
	}
	return nil
}

func testConnectivity(serviceURL string) error {
	healthURL := serviceURL + "/readiness"

	client := &http.Client{
		Timeout:   testTimeout,
		Transport: tlsutils.TransportWithServiceCA(),
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		return errors.Wrap(err, "failed to connect to AI service")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AI service returned status %d", resp.StatusCode)
	}

	return nil
}

func apiToStorage(ai *v2.AiIntegration) *storage.AiIntegration {
	return &storage.AiIntegration{
		Id:         ai.GetId(),
		Name:       ai.GetName(),
		Type:       storage.AiIntegrationType(ai.GetType()),
		ServiceUrl: ai.GetServiceUrl(),
	}
}

func storageToAPI(s *storage.AiIntegration) *v2.AiIntegration {
	return &v2.AiIntegration{
		Id:         s.GetId(),
		Name:       s.GetName(),
		Type:       v2.AiIntegrationType(s.GetType()),
		ServiceUrl: s.GetServiceUrl(),
	}
}
