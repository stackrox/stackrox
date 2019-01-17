package service

import (
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

	v1.ServiceIdentityServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(storage store.Store) Service {
	return &serviceImpl{
		storage: storage,
	}
}
