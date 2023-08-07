package reportgenerator

import (
	"github.com/graph-gophers/graphql-go"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/pkg/notifier"
)

// ReportGenerator interface is used to generate vulnerability report and send notification.
//
//go:generate mockgen-wrapper
type ReportGenerator interface {
	// ProcessReportRequest will generate a report and send notification via the requested notification method.
	// On success, report will be generated and notified, and report snapshot will be stored to the db.
	// On failure, it will log any errors and store it in the report snapshot.
	ProcessReportRequest(req *ReportRequest)
}

// New will create a new instance of the ReportGenerator
func New(
	reportSnapshotStore reportSnapshotDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	watchedImageDatastore watchedImageDS.DataStore,
	collectionQueryResolver collectionDS.QueryResolver,
	notificationProcessor notifier.Processor,
	blobDatastore blobDS.Datastore,
	schema *graphql.Schema,
) ReportGenerator {
	return newReportGeneratorImpl(
		reportSnapshotStore,
		deploymentDatastore,
		watchedImageDatastore,
		collectionQueryResolver,
		notificationProcessor,
		blobDatastore,
		schema,
	)
}

func newReportGeneratorImpl(
	reportSnapshotStore reportSnapshotDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	watchedImageDatastore watchedImageDS.DataStore,
	collectionQueryResolver collectionDS.QueryResolver,
	notificationProcessor notifier.Processor,
	blobStore blobDS.Datastore,
	schema *graphql.Schema,
) *reportGeneratorImpl {
	return &reportGeneratorImpl{
		reportSnapshotStore:     reportSnapshotStore,
		deploymentDatastore:     deploymentDatastore,
		watchedImageDatastore:   watchedImageDatastore,
		collectionQueryResolver: collectionQueryResolver,
		notificationProcessor:   notificationProcessor,
		blobStore:               blobStore,

		Schema: schema,
	}
}
