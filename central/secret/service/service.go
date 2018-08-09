package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/secret/search"
	"github.com/stackrox/rox/central/secret/store"
	"github.com/stackrox/rox/generated/api/v1"
	"google.golang.org/grpc"
)

// Service provides the interface to the microservice that serves secret data.
type Service interface {
	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetSecret(ctx context.Context, request *v1.ResourceByID) (*v1.SecretAndRelationship, error)
	GetSecrets(ctx context.Context, request *v1.RawQuery) (*v1.SecretAndRelationshipList, error)
}

// New returns a new Service instance using the given DB and index.
func New(storage store.Store, searcher search.Searcher) Service {
	return &serviceImpl{
		storage:  storage,
		searcher: searcher,
	}
}
