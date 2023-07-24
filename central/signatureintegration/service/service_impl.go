package service

import (
	"context"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/signatureintegration/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Integration)): {
			"/v1.SignatureIntegrationService/ListSignatureIntegrations",
			"/v1.SignatureIntegrationService/GetSignatureIntegration",
		},
		user.With(permissions.Modify(resources.Integration)): {
			"/v1.SignatureIntegrationService/PostSignatureIntegration",
			"/v1.SignatureIntegrationService/PutSignatureIntegration",
			"/v1.SignatureIntegrationService/DeleteSignatureIntegration",
		},
	})
)

type serviceImpl struct {
	v1.UnimplementedSignatureIntegrationServiceServer

	datastore        datastore.DataStore
	reprocessingLoop reprocessor.Loop
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

func (s *serviceImpl) ListSignatureIntegrations(ctx context.Context, _ *v1.Empty) (*v1.ListSignatureIntegrationsResponse, error) {
	integrations, err := s.datastore.GetAllSignatureIntegrations(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve signature integrations")
	}

	// List integrations in the same order for consistency across requests.
	// Names are unique, thus we don't have to use sort.SliceStable
	sort.Slice(integrations, func(i, j int) bool {
		return integrations[i].GetName() < integrations[j].GetName()
	})
	return &v1.ListSignatureIntegrationsResponse{
		Integrations: integrations,
	}, nil
}

func (s *serviceImpl) GetSignatureIntegration(ctx context.Context, id *v1.ResourceByID) (*storage.SignatureIntegration, error) {
	integration, found, err := s.datastore.GetSignatureIntegration(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve signature integration")
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "failed to retrieve signature integration %q", id.GetId())
	}
	return integration, nil
}

func (s *serviceImpl) PostSignatureIntegration(ctx context.Context, requestedIntegration *storage.SignatureIntegration) (*storage.SignatureIntegration, error) {
	integration, err := s.datastore.AddSignatureIntegration(ctx, requestedIntegration)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create signature integration")
	}
	s.reprocessingLoop.ReprocessSignatureVerifications()
	return integration, nil
}

func (s *serviceImpl) PutSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (*v1.Empty, error) {
	hasUpdatedKeys, err := s.datastore.UpdateSignatureIntegration(ctx, integration)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update signature integration")
	}

	// Only trigger reprocessing of signature verification results when the keys have been updated.
	if hasUpdatedKeys {
		s.reprocessingLoop.ReprocessSignatureVerifications()
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteSignatureIntegration(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	err := s.datastore.RemoveSignatureIntegration(ctx, id.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "failed to delete signature integration")
	}
	s.reprocessingLoop.ReprocessSignatureVerifications()
	return &v1.Empty{}, nil
}
