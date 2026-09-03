package node

import (
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	nodeCVEDS "github.com/stackrox/rox/central/cve/node/datastore"
	"github.com/stackrox/rox/central/globaldb"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	rg   reportGen.ReportGenerator
)

func initialize() {
	if !features.NodeVulnerabilityReports.Enabled() {
		return
	}
	rg = New(
		globaldb.GetPostgres(),
		reportSnapshotDS.Singleton(),
		notifierProcessor.Singleton(),
		blobDS.Singleton(),
		clusterDS.Singleton(),
		namespaceDS.Singleton(),
		nodeCVEDS.Singleton(),
	)
}

// Singleton returns a singleton instance of the node ReportGenerator.
// Returns nil when the NodeVulnerabilityReports feature flag is disabled.
func Singleton() reportGen.ReportGenerator {
	once.Do(initialize)
	return rg
}
