package service

import (
	"context"

	"github.com/stackrox/rox/central/deployment/datastore"
	multiplierStore "github.com/stackrox/rox/central/multiplier/store"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processWhitelistDataStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	processWhitelistResultsStore "github.com/stackrox/rox/central/processwhitelistresults/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
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

	v1.DeploymentServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore, processIndicators processIndicatorDataStore.DataStore, processWhitelists processWhitelistDataStore.DataStore,
	processWhitelistResults processWhitelistResultsStore.DataStore, multipliers multiplierStore.Store, manager riskManager.Manager) Service {
	return &serviceImpl{
		datastore:               datastore,
		processIndicators:       processIndicators,
		processWhitelists:       processWhitelists,
		processWhitelistResults: processWhitelistResults,
		multipliers:             multipliers,
		manager:                 manager,
	}
}
