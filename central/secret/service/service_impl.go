package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/role/resources"
	"bitbucket.org/stack-rox/apollo/central/secret/search"
	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/auth/permissions"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/perrpc"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Secret)): {
			"/v1.SecretService/GetSecret",
			"/v1.SecretService/GetSecrets",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	storage  store.Store
	searcher search.Searcher
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSecretServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterSecretServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(authorizer.Authorized(ctx, fullMethodName))
}

// GetSecret returns the secret for the id.
func (s *serviceImpl) GetSecret(ctx context.Context, request *v1.ResourceByID) (*v1.SecretAndRelationship, error) {
	secret, exists, err := s.storage.GetSecret(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "secret with id '%s' does not exist", request.GetId())
	}

	relationship, exists, err := s.storage.GetRelationship(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "relationship with id '%s' does not exist", request.GetId())
	}

	return &v1.SecretAndRelationship{
		Secret:       secret,
		Relationship: relationship,
	}, nil
}

// GetSecrets returns all secrets that match the query.
func (s *serviceImpl) GetSecrets(ctx context.Context, rawQuery *v1.RawQuery) (*v1.SecretAndRelationshipList, error) {
	sars, err := s.searcher.SearchRawSecrets(rawQuery)
	if err != nil {
		return nil, err
	}
	return &v1.SecretAndRelationshipList{SecretAndRelationships: sars}, nil
}
