package manager

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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

	TriggerRun(clusterID, standardID string) (*v1.ComplianceRun, error)
}

// NewManager creates and returns a new compliance manager.
func NewManager(scheduleStore ScheduleStore, clusterStore clusterDatastore.DataStore, nodeStore store.GlobalStore, deploymentStore datastore.DataStore) (ComplianceManager, error) {
	return newManager(scheduleStore, clusterStore, nodeStore, deploymentStore)
}
