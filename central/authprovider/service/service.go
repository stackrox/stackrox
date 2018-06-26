package service

import (
	"context"

	"bitbucket.org/stack-rox/apollo/central/authprovider/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/authproviders"
	"bitbucket.org/stack-rox/apollo/pkg/grpc"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/golang/protobuf/ptypes/empty"
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

// AuthProviderUpdater knows how to emplace or remove auth providers.
type authProviderUpdater interface {
	UpdateProvider(id string, provider authproviders.Authenticator)
	RemoveProvider(id string)
}

// New returns a new Service instance using the given DataStore.
func New(storage store.Store, auth authProviderUpdater) Service {
	return &serviceImpl{
		storage: storage,
		auth:    auth,
	}
}
