package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Secret)): {
			"/v1.SecretService/GetSecret",
			"/v1.SecretService/ListSecrets",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	storage     datastore.DataStore
	deployments deploymentDatastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSecretServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterSecretServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetSecret returns the secret for the id.
func (s *serviceImpl) GetSecret(ctx context.Context, request *v1.ResourceByID) (*v1.Secret, error) {
	secret, exists, err := s.storage.GetSecret(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "secret with id '%s' does not exist", request.GetId())
	}

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, secret.GetClusterId()).
		AddExactMatches(search.Namespace, secret.GetNamespace()).
		AddExactMatches(search.SecretName, secret.GetName()).
		ProtoQuery()

	deploymentResults, err := s.deployments.SearchDeployments(psr)
	if err != nil {
		return nil, err
	}

	var deployments []*v1.SecretDeploymentRelationship
	for _, r := range deploymentResults {
		deployments = append(deployments, &v1.SecretDeploymentRelationship{
			Id:   r.Id,
			Name: r.Name,
		})
	}
	secret.Relationship = &v1.SecretRelationship{
		DeploymentRelationships: deployments,
	}
	return secret, nil
}

// ListSecrets returns all secrets that match the query.
func (s *serviceImpl) ListSecrets(ctx context.Context, rawQuery *v1.RawQuery) (*v1.ListSecretsResponse, error) {
	q, err := search.ParseRawQueryOrEmpty(rawQuery.GetQuery())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	secrets, err := s.storage.SearchListSecrets(q)
	if err != nil {
		return nil, err
	}
	return &v1.ListSecretsResponse{Secrets: secrets}, nil
}
