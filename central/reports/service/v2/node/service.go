package node

import (
	"context"

	blobDS "github.com/stackrox/rox/central/blob/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/postgres"
)

// Service provides the interface to the gRPC service for node vulnerability reports.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	apiV2.NodeReportServiceServer
}

// New returns a new instance of the node report service.
func New(reportConfigStore reportConfigDS.DataStore, snapshotDatastore snapshotDS.DataStore,
	collectionDatastore collectionDS.DataStore, notifierDatastore notifierDS.DataStore,
	scheduler schedulerV2.Scheduler, blobStore blobDS.Datastore, validator *validation.Validator,
	db postgres.DB) Service {
	return &serviceImpl{
		reportConfigStore:   reportConfigStore,
		snapshotDatastore:   snapshotDatastore,
		collectionDatastore: collectionDatastore,
		notifierDatastore:   notifierDatastore,
		scheduler:           scheduler,
		blobStore:           blobStore,
		validator:           validator,
		db:                  db,
	}
}
