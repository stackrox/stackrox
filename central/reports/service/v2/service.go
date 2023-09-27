package v2

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
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for reports.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	apiV2.ReportServiceServer
}

// New returns a new instance of the service.
func New(reportConfigStore reportConfigDS.DataStore, snapshotDatastore snapshotDS.DataStore,
	collectionDatastore collectionDS.DataStore, notifierDatastore notifierDS.DataStore,
	scheduler schedulerV2.Scheduler, blobStore blobDS.Datastore, validator *validation.Validator) Service {
	if !features.VulnReportingEnhancements.Enabled() {
		return nil
	}
	return &serviceImpl{
		reportConfigStore:   reportConfigStore,
		snapshotDatastore:   snapshotDatastore,
		collectionDatastore: collectionDatastore,
		notifierDatastore:   notifierDatastore,
		scheduler:           scheduler,
		blobStore:           blobStore,
		validator:           validator,
	}
}
