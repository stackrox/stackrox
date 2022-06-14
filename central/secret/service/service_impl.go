package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	deploymentDatastore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/central/secret/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stackrox/stackrox/pkg/grpc/authz"
	"github.com/stackrox/stackrox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/stackrox/pkg/grpc/authz/user"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/paginated"
	"google.golang.org/grpc"
)

const (
	maxSecretsReturned = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Secret)): {
			"/v1.SecretService/GetSecret",
			"/v1.SecretService/CountSecrets",
			"/v1.SecretService/ListSecrets",
		},
	})
)

// serviceImpl provides APIs for alerts.
type serviceImpl struct {
	secrets     datastore.DataStore
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
func (s *serviceImpl) GetSecret(ctx context.Context, request *v1.ResourceByID) (*storage.Secret, error) {
	secret, exists, err := s.secrets.GetSecret(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "secret with id '%s' does not exist", request.GetId())
	}

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, secret.GetClusterId()).
		AddExactMatches(search.Namespace, secret.GetNamespace()).
		AddExactMatches(search.SecretName, secret.GetName()).
		ProtoQuery()

	deploymentResults, err := s.deployments.SearchDeployments(ctx, psr)
	if err != nil {
		return nil, err
	}

	var deployments []*storage.SecretDeploymentRelationship
	for _, r := range deploymentResults {
		deployments = append(deployments, &storage.SecretDeploymentRelationship{
			Id:   r.Id,
			Name: r.Name,
		})
	}
	secret.Relationship = &storage.SecretRelationship{
		DeploymentRelationships: deployments,
	}
	return secret, nil
}

// CountSecrets counts the number of secrets that match the input query.
func (s *serviceImpl) CountSecrets(ctx context.Context, request *v1.RawQuery) (*v1.CountSecretsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	numSecrets, err := s.secrets.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.CountSecretsResponse{Count: int32(numSecrets)}, nil
}

// ListSecrets returns all secrets that match the query.
func (s *serviceImpl) ListSecrets(ctx context.Context, request *v1.RawQuery) (*v1.ListSecretsResponse, error) {
	// Fill in query.
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, request.Pagination, maxSecretsReturned)

	secrets, err := s.secrets.SearchListSecrets(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.ListSecretsResponse{Secrets: secrets}, nil
}
