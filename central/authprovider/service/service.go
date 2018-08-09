package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stackrox/rox/central/authprovider/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetAuthProvider(ctx context.Context, request *v1.ResourceByID) (*v1.AuthProvider, error)
	GetAuthProviders(ctx context.Context, request *v1.GetAuthProvidersRequest) (*v1.GetAuthProvidersResponse, error)
	PostAuthProvider(ctx context.Context, request *v1.AuthProvider) (*v1.AuthProvider, error)
	PutAuthProvider(ctx context.Context, request *v1.AuthProvider) (*empty.Empty, error)
	DeleteAuthProvider(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(storage store.Store) Service {
	return &serviceImpl{
		storage: storage,
	}
}
