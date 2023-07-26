package v2

import (
	"context"

	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reportconfigurations/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for roles.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	apiV2.ReportConfigurationServiceServer
}

// New returns a new instance of the service. Please use the Singleton instead.
func New(reportConfigStore datastore.DataStore,
	notifierDatastore notifierDataStore.DataStore,
	collectionDatastore collectionDataStore.DataStore,
	scheduler schedulerV2.Scheduler) Service {
	return &serviceImpl{
		scheduler:           scheduler,
		reportConfigStore:   reportConfigStore,
		collectionDatastore: collectionDatastore,
		notifierDatastore:   notifierDatastore,
	}
}
