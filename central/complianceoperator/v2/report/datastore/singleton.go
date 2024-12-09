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
	"github.com/stackrox/rox/pkg/search"
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

	// Purge any orphan report snapshots (reports that were not completed)
	// This can only happen if central is restarted while reports are still running.
	// Sensor will send the Scan and CheckResult resources again and Central will
	// regenerate the Reports, so there isn't any data loss by doing this.
	searchQuery := search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorReportState,
			storage.ComplianceOperatorReportStatus_WAITING.String(),
			storage.ComplianceOperatorReportStatus_PREPARING.String(),
			storage.ComplianceOperatorReportStatus_GENERATED.String(),
		).ProtoQuery()
	orphanSnapshots, err := dataStore.SearchSnapshots(complianceOperatorSnapshotAdministrationCtx, searchQuery)
	if err != nil {
		log.Errorf("unable to search for orphan snapshots: %v", orphanSnapshots)
	}
	for _, snapshot := range orphanSnapshots {
		if err := dataStore.DeleteSnapshot(complianceOperatorSnapshotAdministrationCtx, snapshot.GetReportId()); err != nil {
			log.Errorf("unable to delete snapshot %s", snapshot.GetReportId())
		}
	}
}
