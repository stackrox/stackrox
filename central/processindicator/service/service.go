package service

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	baselineDataStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	v1.ProcessServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(processIndicators processIndicatorDataStore.DataStore, deployments deploymentDataStore.DataStore, baselines baselineDataStore.DataStore) Service {
	return &serviceImpl{
		deployments:       deployments,
		processIndicators: processIndicators,
		baselines:         baselines,
	}
}
