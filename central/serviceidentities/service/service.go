package service

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stackrox/rox/central/serviceidentities/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/net/context"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetServiceIdentities(ctx context.Context, _ *empty.Empty) (*v1.ServiceIdentityResponse, error)
	CreateServiceIdentity(ctx context.Context, request *v1.CreateServiceIdentityRequest) (*v1.CreateServiceIdentityResponse, error)
	GetAuthorities(ctx context.Context, request *empty.Empty) (*v1.Authorities, error)
}

// New returns a new Service instance using the given DataStore.
func New(storage store.Store) Service {
	return &serviceImpl{
		storage: storage,
	}
}
