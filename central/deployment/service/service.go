package service

import (
	"context"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	multiplierStore "github.com/stackrox/rox/central/multiplier/store"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/api/v1"
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
func New(datastore datastore.DataStore, processIndicators processIndicatorDataStore.DataStore, multipliers multiplierStore.Store, enricher enrichment.Enricher) Service {
	return &serviceImpl{
		datastore:         datastore,
		processIndicators: processIndicators,
		multipliers:       multipliers,
		enricher:          enricher,
	}
}
