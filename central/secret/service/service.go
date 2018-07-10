package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/secret/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
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
func New(storage store.Store, index bleve.Index) Service {
	return &serviceImpl{
		storage: storage,
		index:   index,
	}
}
