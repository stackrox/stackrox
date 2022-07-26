package service

import (
	"context"

	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
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

	v1.AuthProviderServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(registry authproviders.Registry, groupStore groupDataStore.DataStore) Service {
	return &serviceImpl{
		registry:   registry,
		groupStore: groupStore,
	}
}

// NewWithBasicProviderDisabled returns a new Service instance using the given DataStore.
// The service would filter out "basic" auth provider.
func NewWithBasicProviderDisabled(registry authproviders.Registry, groupStore groupDataStore.DataStore) Service {
	return &basicAuthProviderRemovalServiceImpl{
		underlying: New(registry, groupStore),
	}
}
