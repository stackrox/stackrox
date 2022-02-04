package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/signature/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.SignatureIntegration)): {
			"/v1.SignatureIntegrationService/GetSignatureIntegrations",
			"/v1.SignatureIntegrationService/GetSignatureIntegration",
		},
		user.With(permissions.Modify(resources.SignatureIntegration)): {
			"/v1.SignatureIntegrationService/PostSignatureIntegration",
			"/v1.SignatureIntegrationService/PutSignatureIntegration",
			"/v1.SignatureIntegrationService/DeleteSignatureIntegration",
		},
	})
	signatureIntegrationIDPrefix = "io.stackrox.authz.signatureintegration."
)

type serviceImpl struct {
	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSignatureIntegrationServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterSignatureIntegrationServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetSignatureIntegrations(ctx context.Context, _ *v1.Empty) (*v1.GetSignatureIntegrationsResponse, error) {
	integrations, err := s.datastore.GetSignatureIntegrations(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve signature integrations")
	}
	// List integrations in the same order for consistency across requests.
	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].GetName() < integrations[j].GetName()
	})
	return &v1.GetSignatureIntegrationsResponse{
		Integrations: integrations,
	}, nil
}

func (s *serviceImpl) GetSignatureIntegration(ctx context.Context, id *v1.ResourceByID) (*storage.SignatureIntegration, error) {
	signatureIntegration, found, err := s.datastore.GetSignatureIntegration(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve signature integration %s", id.GetId())
	}
	if !found {
		return nil, errors.Wrapf(errorhelpers.ErrNotFound, "Signature integration %s not found", id.GetId())
	}
	return signatureIntegration, nil
}

func (s *serviceImpl) PostSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (*storage.SignatureIntegration, error) {
	integration.Id = generateSignatureIntegrationID()
	if err := validateSignatureIntegration(integration); err != nil {
		return nil, err
	}
	err := s.datastore.AddSignatureIntegration(ctx, integration)
	if err != nil {
		return nil, err
	}

	return integration, nil
}

func (s *serviceImpl) PutSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (*v1.Empty, error) {
	if err := validateSignatureIntegration(integration); err != nil {
		return nil, err
	}
	err := s.datastore.UpdateSignatureIntegration(ctx, integration)
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteSignatureIntegration(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	err := s.datastore.RemoveSignatureIntegration(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to delete signature integration %s", id.GetId())
	}
	return &v1.Empty{}, nil
}

func generateSignatureIntegrationID() string {
	return signatureIntegrationIDPrefix + uuid.NewV4().String()
}

func validateSignatureIntegration(integration *storage.SignatureIntegration) error {
	if integration.GetName() == "" {
		return errors.New("name is not specified for integration")
	}
	if len(integration.GetSignatureVerificationConfigs()) == 0 {
		return errors.New("integration should have at least one signature verification config")
	}
	return nil
}
