package reportgenerator

import (
	"github.com/graph-gophers/graphql-go"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	reportMetadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
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
func New(reportConfigDatastore reportConfigDS.DataStore,
	reportMetadataStore reportMetadataDS.DataStore,
	reportSnapshotStore reportSnapshotDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	watchedImageDatastore watchedImageDS.DataStore,
	collectionDatastore collectionDS.DataStore,
	collectionQueryResolver collectionDS.QueryResolver,
	notifierDatastore notifierDS.DataStore,
	notificationProcessor notifier.Processor,
	schema *graphql.Schema,
) ReportGenerator {
	return newReportGeneratorImpl(
		reportConfigDatastore,
		reportMetadataStore,
		reportSnapshotStore,
		deploymentDatastore,
		watchedImageDatastore,
		collectionDatastore,
		collectionQueryResolver,
		notifierDatastore,
		notificationProcessor,
		schema,
	)
}

func newReportGeneratorImpl(reportConfigDatastore reportConfigDS.DataStore,
	reportMetadataStore reportMetadataDS.DataStore,
	reportSnapshotStore reportSnapshotDS.DataStore,
	deploymentDatastore deploymentDS.DataStore,
	watchedImageDatastore watchedImageDS.DataStore,
	collectionDatastore collectionDS.DataStore,
	collectionQueryResolver collectionDS.QueryResolver,
	notifierDatastore notifierDS.DataStore,
	notificationProcessor notifier.Processor,
	schema *graphql.Schema,
) *reportGeneratorImpl {
	return &reportGeneratorImpl{
		reportConfigDatastore:   reportConfigDatastore,
		reportMetadataStore:     reportMetadataStore,
		reportSnapshotStore:     reportSnapshotStore,
		deploymentDatastore:     deploymentDatastore,
		watchedImageDatastore:   watchedImageDatastore,
		collectionDatastore:     collectionDatastore,
		collectionQueryResolver: collectionQueryResolver,
		notifierDatastore:       notifierDatastore,
		notificationProcessor:   notificationProcessor,

		Schema: schema,
	}
}
