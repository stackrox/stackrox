package service

import (
	"context"

	"github.com/stackrox/rox/central/processwhitelist/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service is the interface to the gRPC service for managing process whitelists
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ProcessWhitelistServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(store datastore.DataStore, riskManager riskManager.Manager) Service {
	return &serviceImpl{
		dataStore:   store,
		riskManager: riskManager,
	}
}
