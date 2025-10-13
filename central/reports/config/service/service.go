package service

import (
	"context"

	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/central/reports/manager"
	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for roles.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	v1.ReportConfigurationServiceServer
}

// New returns a new instance of the service. Please use the Singleton instead.
func New(reportConfigStore datastore.DataStore, notifierStore notifierDataStore.DataStore,
	collectionDatastore collectionDataStore.DataStore, manager manager.Manager) Service {
	return &serviceImpl{
		manager:             manager,
		reportConfigStore:   reportConfigStore,
		collectionDatastore: collectionDatastore,
		notifierDatastore:   notifierStore,
	}
}
