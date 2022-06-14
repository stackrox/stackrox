package service

import (
	"context"

	groupDataStore "github.com/stackrox/stackrox/central/group/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/auth/authproviders"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
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
