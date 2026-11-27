package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/report/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	dataStore DataStore

	complianceOperatorSnapshotAdministrationCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))

	log = logging.LoggerForModule()
)

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return dataStore
}

func initialize() {
	store := postgres.New(globaldb.GetPostgres())
	dataStore = New(store)

	// Purge any orphan report snapshots (reports that were not completed): this happens when
	// Central restarts while reports are still running. Pending state must not survive a
	// restart, because nothing resumes it: scheduled reports are re-created by the next
	// scheduled cycle and on-demand reports are re-requestable. The decoupled report
	// scheduler relies on this purge as well (see DeleteOrphanedReportSnapshots).
	if err := DeleteOrphanedReportSnapshots(complianceOperatorSnapshotAdministrationCtx, dataStore); err != nil {
		log.Errorf("DeleteOrphanedReportSnapshots failed: %v", err)
	}
}
