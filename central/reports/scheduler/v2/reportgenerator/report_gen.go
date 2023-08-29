package reportgenerator

import (
	"github.com/graph-gophers/graphql-go"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/postgres"
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
	db postgres.DB,
	reportSnapshotStore reportSnapshotDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	watchedImageDatastore watchedImageDS.DataStore,
	collectionQueryResolver collectionDS.QueryResolver,
	notificationProcessor notifier.Processor,
	blobDatastore blobDS.Datastore,
	clusterDatastore clusterDS.DataStore,
	namespaceDatastore namespaceDS.DataStore,
	schema *graphql.Schema,
) ReportGenerator {
	return newReportGeneratorImpl(
		db,
		reportSnapshotStore,
		deploymentDatastore,
		watchedImageDatastore,
		collectionQueryResolver,
		notificationProcessor,
		blobDatastore,
		clusterDatastore,
		namespaceDatastore,
		schema,
	)
}

func newReportGeneratorImpl(
	db postgres.DB,
	reportSnapshotStore reportSnapshotDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	watchedImageDatastore watchedImageDS.DataStore,
	collectionQueryResolver collectionDS.QueryResolver,
	notificationProcessor notifier.Processor,
	blobStore blobDS.Datastore,
	clusterDatastore clusterDS.DataStore,
	namespaceDatastore namespaceDS.DataStore,
	schema *graphql.Schema,
) *reportGeneratorImpl {
	return &reportGeneratorImpl{
		reportSnapshotStore:     reportSnapshotStore,
		deploymentDatastore:     deploymentDatastore,
		watchedImageDatastore:   watchedImageDatastore,
		collectionQueryResolver: collectionQueryResolver,
		notificationProcessor:   notificationProcessor,
		clusterDatastore:        clusterDatastore,
		namespaceDatastore:      namespaceDatastore,
		blobStore:               blobStore,
		db:                      db,

		Schema: schema,
	}
}
