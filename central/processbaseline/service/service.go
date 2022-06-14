package service

import (
	"context"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service is the interface to the gRPC service for managing process baselines
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ProcessBaselineServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(store datastore.DataStore, reprocessor reprocessor.Loop, connectionManager connection.Manager, deployments deploymentDataStore.DataStore, lifecycleManager lifecycle.Manager) Service {
	return &serviceImpl{
		dataStore:         store,
		reprocessor:       reprocessor,
		connectionManager: connectionManager,
		deployments:       deployments,
		lifecycleManager:  lifecycleManager,
	}
}
