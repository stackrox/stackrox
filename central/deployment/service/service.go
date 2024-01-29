package service

import (
	"context"

	"github.com/stackrox/rox/central/deployment/datastore"
	processBaselineDataStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processBaselineResultsStore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves deployment data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.DeploymentServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore, processIndicators processIndicatorDataStore.DataStore, processBaselines processBaselineDataStore.DataStore,
	processBaselineResults processBaselineResultsStore.DataStore, risks riskDataStore.DataStore, manager riskManager.Manager) Service {
	return &serviceImpl{
		datastore:              datastore,
		processIndicators:      processIndicators,
		processBaselines:       processBaselines,
		processBaselineResults: processBaselineResults,
		risks:                  risks,
		manager:                manager,
	}
}
