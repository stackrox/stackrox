package manager

import (
	"context"

	clusterDatastore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/compliance"
	"github.com/stackrox/stackrox/central/compliance/data"
	complianceDS "github.com/stackrox/stackrox/central/compliance/datastore"
	"github.com/stackrox/stackrox/central/compliance/standards"
	complianceOperatorCheckDS "github.com/stackrox/stackrox/central/complianceoperator/checkresults/datastore"
	complianceOperatorManager "github.com/stackrox/stackrox/central/complianceoperator/manager"
	"github.com/stackrox/stackrox/central/deployment/datastore"
	nodeDatastore "github.com/stackrox/stackrox/central/node/globaldatastore"
	podDatastore "github.com/stackrox/stackrox/central/pod/datastore"
	"github.com/stackrox/stackrox/central/scrape/factory"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
)

const (
	// Wildcard is a special string that indicates a check should be run for all clusters/standards. Use
	// `ExpandSelection` to expand it to a list of cluster/standard pairs.
	Wildcard = "*"
)

// ComplianceManager manages compliance schedules and one-off compliance runs.
type ComplianceManager interface {
	Start() error
	Stop() error

	GetSchedules(ctx context.Context, request *v1.GetComplianceRunSchedulesRequest) ([]*v1.ComplianceRunScheduleInfo, error)
	AddSchedule(ctx context.Context, spec *storage.ComplianceRunSchedule) (*v1.ComplianceRunScheduleInfo, error)
	UpdateSchedule(ctx context.Context, spec *storage.ComplianceRunSchedule) (*v1.ComplianceRunScheduleInfo, error)
	DeleteSchedule(ctx context.Context, id string) error

	GetRecentRuns(ctx context.Context, request *v1.GetRecentComplianceRunsRequest) ([]*v1.ComplianceRun, error)
	GetRecentRun(ctx context.Context, id string) (*v1.ComplianceRun, error)

	ExpandSelection(ctx context.Context, clusterIDOrWildcard, standardIDOrWildcard string) ([]compliance.ClusterStandardPair, error)

	TriggerRuns(ctx context.Context, clusterStandardPairs ...compliance.ClusterStandardPair) ([]*v1.ComplianceRun, error)

	// GetRunStatuses returns the statuses for the runs with the given IDs. Any runs that could not be located (e.g.,
	// because they are too old or the ID is invalid) will be returned in the id to error map.
	GetRunStatuses(ctx context.Context, ids ...string) ([]*v1.ComplianceRun, error)
}

// NewManager creates and returns a new compliance manager.
func NewManager(standardsRegistry *standards.Registry,
	complianceOperatorManager complianceOperatorManager.Manager,
	complianceOperatorResults complianceOperatorCheckDS.DataStore,
	scheduleStore ScheduleStore,
	clusterStore clusterDatastore.DataStore,
	nodeStore nodeDatastore.GlobalDataStore,
	deploymentStore datastore.DataStore,
	podStore podDatastore.DataStore,
	dataRepoFactory data.RepositoryFactory,
	scrapeFactory factory.ScrapeFactory,
	resultsStore complianceDS.DataStore) (ComplianceManager, error) {
	return newManager(standardsRegistry, complianceOperatorManager, complianceOperatorResults, scheduleStore, clusterStore, nodeStore, deploymentStore, podStore, dataRepoFactory, scrapeFactory, resultsStore)
}
