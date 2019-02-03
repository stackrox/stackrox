package manager

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/data"
	complianceResultsStore "github.com/stackrox/rox/central/compliance/store"
	"github.com/stackrox/rox/central/deployment/datastore"
	nodeStore "github.com/stackrox/rox/central/node/globalstore"
	"github.com/stackrox/rox/central/scrape"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

	GetSchedules(request *v1.GetComplianceRunSchedulesRequest) []*v1.ComplianceRunScheduleInfo
	GetSchedule(id string) (*v1.ComplianceRunScheduleInfo, error)
	AddSchedule(spec *storage.ComplianceRunSchedule) (*v1.ComplianceRunScheduleInfo, error)
	UpdateSchedule(spec *storage.ComplianceRunSchedule) (*v1.ComplianceRunScheduleInfo, error)
	DeleteSchedule(id string) error

	GetRecentRuns(request *v1.GetRecentComplianceRunsRequest) []*v1.ComplianceRun
	GetRecentRun(id string) (*v1.ComplianceRun, error)

	ExpandSelection(clusterIDOrWildcard, standardIDOrWildcard string) ([]compliance.ClusterStandardPair, error)

	TriggerRuns(clusterStandardPairs ...compliance.ClusterStandardPair) ([]*v1.ComplianceRun, error)

	// GetRunStatuses returns the statuses for the runs with the given IDs. Any runs that could not be located (e.g.,
	// because they are too old or the ID is invalid) will be returned in the id to error map.
	GetRunStatuses(ids ...string) []*v1.ComplianceRun
}

// NewManager creates and returns a new compliance manager.
func NewManager(standardImplStore StandardImplementationStore, scheduleStore ScheduleStore, clusterStore clusterDatastore.DataStore, nodeStore nodeStore.GlobalStore, deploymentStore datastore.DataStore, dataRepoFactory data.RepositoryFactory, scrapeFactory scrape.Factory, resultsStore complianceResultsStore.Store) (ComplianceManager, error) {
	return newManager(standardImplStore, scheduleStore, clusterStore, nodeStore, deploymentStore, dataRepoFactory, scrapeFactory, resultsStore)
}
