package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/logintegrations/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logintegration"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = or.SensorOrAuthorizer(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.LogIntegration)): {
			"/v1.LogIntegrationService/GetLogIntegrations",
			"/v1.LogIntegrationService/GetLogIntegration",
		},
		user.With(permissions.Modify(resources.LogIntegration)): {
			"/v1.LogIntegrationService/CreateLogIntegration",
			"/v1.LogIntegrationService/UpdateLogIntegration",
			"/v1.LogIntegrationService/DeleteLogIntegration",
			"/v1.LogIntegrationService/TestLogIntegration",
			"/v1.LogIntegrationService/TestUpdatedLogIntegration",
		},
	}))
)

type serviceImpl struct {
	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterLogIntegrationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterLogIntegrationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetLogIntegration(ctx context.Context, req *v1.ResourceByID) (*v1.GetLogIntegrationResponse, error) {
	if features.K8sAuditLogDetection.Enabled() {
		return nil, status.Error(codes.Unimplemented, logintegration.ErrFeatureNotEnabled)
	}

	integration, found, err := s.datastore.GetLogIntegration(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, logintegration.ErrNotFound, req.GetId())
	}
	return &v1.GetLogIntegrationResponse{
		LogIntegration: integration,
	}, nil
}

func (s *serviceImpl) GetLogIntegrations(ctx context.Context, _ *v1.Empty) (*v1.GetLogIntegrationsResponse, error) {
	if features.K8sAuditLogDetection.Enabled() {
		return nil, status.Error(codes.Unimplemented, logintegration.ErrFeatureNotEnabled)
	}

	integrations, err := s.datastore.GetLogIntegrations(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.GetLogIntegrationsResponse{
		Integrations: integrations,
	}, nil
}

func (s *serviceImpl) CreateLogIntegration(ctx context.Context, req *v1.CreateLogIntegrationRequest) (*v1.CreateLogIntegrationResponse, error) {
	if features.K8sAuditLogDetection.Enabled() {
		return nil, status.Error(codes.Unimplemented, logintegration.ErrFeatureNotEnabled)
	}

	if req.GetLogIntegration().GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "ID not expected in create Log Integration request")
	}
	req.LogIntegration.Id = uuid.NewV4().String()

	if err := s.datastore.CreateLogIntegration(ctx, req.GetLogIntegration()); err != nil {
		return nil, err
	}
	return &v1.CreateLogIntegrationResponse{
		LogIntegration: req.GetLogIntegration(),
	}, nil
}

func (s *serviceImpl) TestLogIntegration(ctx context.Context, req *v1.TestLogIntegrationRequest) (*v1.Empty, error) {
	if features.K8sAuditLogDetection.Enabled() {
		return nil, status.Error(codes.Unimplemented, logintegration.ErrFeatureNotEnabled)
	}

	return nil, status.Error(codes.Unimplemented, "test log integration not implemented")
}

func (s *serviceImpl) DeleteLogIntegration(ctx context.Context, req *v1.ResourceByID) (*v1.Empty, error) {
	if features.K8sAuditLogDetection.Enabled() {
		return nil, status.Error(codes.Unimplemented, logintegration.ErrFeatureNotEnabled)
	}

	if err := s.datastore.DeleteLogIntegration(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) UpdateLogIntegration(ctx context.Context, req *v1.UpdateLogIntegrationRequest) (*v1.Empty, error) {
	if features.K8sAuditLogDetection.Enabled() {
		return nil, status.Error(codes.Unimplemented, logintegration.ErrFeatureNotEnabled)
	}

	if err := s.reconcileUpdateLogIntegrationRequest(ctx, req); err != nil {
		return nil, err
	}

	if err := s.datastore.UpdateLogIntegration(ctx, req.GetConfig()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) TestUpdatedLogIntegration(ctx context.Context, req *v1.UpdateLogIntegrationRequest) (*v1.Empty, error) {
	if features.K8sAuditLogDetection.Enabled() {
		return nil, status.Error(codes.Unimplemented, logintegration.ErrFeatureNotEnabled)
	}

	if err := s.reconcileUpdateLogIntegrationRequest(ctx, req); err != nil {
		return nil, err
	}

	return nil, status.Error(codes.Unimplemented, "test log integration not implemented")
}

func (s *serviceImpl) reconcileUpdateLogIntegrationRequest(ctx context.Context, updateRequest *v1.UpdateLogIntegrationRequest) error {
	if updateRequest.GetUpdatePassword() {
		return nil
	}

	if updateRequest.GetConfig() == nil {
		return status.Error(codes.InvalidArgument, "request is missing log integration configuration")
	}
	if updateRequest.GetConfig().GetId() == "" {
		return status.Error(codes.InvalidArgument, "id required for stored credential reconciliation")
	}

	existing, _, err := s.datastore.GetLogIntegration(ctx, updateRequest.GetConfig().GetId())
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if existing == nil {
		return status.Errorf(codes.NotFound, logintegration.ErrNotFound, updateRequest.GetConfig().GetId())
	}

	if err := reconcileLogIntegrationWithExisting(updateRequest.GetConfig(), existing); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

func reconcileLogIntegrationWithExisting(update *storage.LogIntegration, existing *storage.LogIntegration) error {
	return secrets.ReconcileScrubbedStructWithExisting(update, existing)
}
