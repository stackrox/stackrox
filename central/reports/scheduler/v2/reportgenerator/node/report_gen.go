package node

import (
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	nodeCVEDS "github.com/stackrox/rox/central/cve/node/datastore"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/postgres"
)

// New creates a new NodeReportGenerator instance.
func New(
	db postgres.DB,
	reportSnapshotStore reportSnapshotDS.DataStore,
	notificationProcessor notifier.Processor,
	blobDatastore blobDS.Datastore,
	clusterDatastore clusterDS.DataStore,
	namespaceDatastore namespaceDS.DataStore,
	nodeCVEDatastore nodeCVEDS.DataStore,
) reportGen.ReportGenerator {
	return &nodeReportGeneratorImpl{
		reportSnapshotStore:   reportSnapshotStore,
		notificationProcessor: notificationProcessor,
		blobStore:             blobDatastore,
		clusterDatastore:      clusterDatastore,
		namespaceDatastore:    namespaceDatastore,
		nodeCVEDatastore:      nodeCVEDatastore,
		db:                    db,
	}
}
