package manager

import (
	"context"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/data"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/standards"
	complianceOperatorCheckDS "github.com/stackrox/rox/central/complianceoperator/checkresults/datastore"
	complianceOperatorManager "github.com/stackrox/rox/central/complianceoperator/manager"
	"github.com/stackrox/rox/central/deployment/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	"github.com/stackrox/rox/central/scrape/factory"
	v1 "github.com/stackrox/rox/generated/api/v1"
)

const (
	// Wildcard is a special string that indicates a check should be run for all clusters/standards. Use
	// `ExpandSelection` to expand it to a list of cluster/standard pairs.
	Wildcard = "*"
)

// ComplianceManager manages compliance schedules and one-off compliance runs.
type ComplianceManager interface {
	GetRecentRuns(ctx context.Context, request *v1.GetRecentComplianceRunsRequest) ([]*v1.ComplianceRun, error)
	GetRecentRun(ctx context.Context, id string) (*v1.ComplianceRun, error)

	ExpandSelection(ctx context.Context, clusterIDOrWildcard, standardIDOrWildcard string) ([]compliance.ClusterStandardPair, error)

	TriggerRuns(ctx context.Context, clusterStandardPairs ...compliance.ClusterStandardPair) ([]*v1.ComplianceRun, error)

	// GetRunStatuses returns the statuses for the runs with the given IDs. Any runs that could not be located (e.g.,
	// because they are too old or the ID is invalid) will be returned in the id to error map.
	GetRunStatuses(ctx context.Context, ids ...string) ([]*v1.ComplianceRun, error)

	// GetLatestRunStatuses returns the statuses for the most recent runs for <cluster, standard> pair. This does not persist
	// across restarts, but neither do run statuses
	GetLatestRunStatuses(ctx context.Context) ([]*v1.ComplianceRun, error)
}

// NewManager creates and returns a new compliance manager.
func NewManager(standardsRegistry *standards.Registry,
	complianceOperatorManager complianceOperatorManager.Manager,
	complianceOperatorResults complianceOperatorCheckDS.DataStore,
	clusterStore clusterDatastore.DataStore,
	nodeStore nodeDatastore.DataStore,
	deploymentStore datastore.DataStore,
	podStore podDatastore.DataStore,
	dataRepoFactory data.RepositoryFactory,
	scrapeFactory factory.ScrapeFactory,
	resultsStore complianceDS.DataStore) ComplianceManager {
	return newManager(standardsRegistry, complianceOperatorManager, complianceOperatorResults, clusterStore, nodeStore, deploymentStore, podStore, dataRepoFactory, scrapeFactory, resultsStore)
}
