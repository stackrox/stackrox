package m214tom215

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_214_to_m_215_populate_alert_deployment_containers_imageidv2/schema"
	alertStore "github.com/stackrox/rox/migrator/migrations/m_214_to_m_215_populate_alert_deployment_containers_imageidv2/store"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

// If so increment the `MinimumSupportedDBVersionSeqNum` to the `CurrentDBVersionSeqNum` of the release immediately
// following the release that cannot tolerate the change in pkg/migrations/internal/fallback_seq_num.go.
//
// For example, in 4.2 a column `column_v2` is added to replace the `column_v1` column in 4.1.
// All the code from 4.2 onward will not reference `column_v1`. At some point in the future a rollback to 4.1
// will not longer be supported and we want to remove `column_v1`. To do so, we will upgrade the schema to remove
// the column and update the `MinimumSupportedDBVersionSeqNum` to be the value of `CurrentDBVersionSeqNum` in 4.2
// as 4.1 will no longer be supported. The migration process will inform the user of an error when trying to migrate
// to a software version that can no longer be supported by the database.

func migrate(database *types.Databases) error {
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTableAlertsStmt)

	db := database.PostgresDB
	store := alertStore.New(db)

	page := 0
	pageSize := 10000
	for {
		batch := make([]*storage.Alert, 0, pageSize)
		pagination := search.NewPagination().Limit(int32(pageSize)).Offset(int32(page * pageSize))
		query := search.NewQueryBuilder().AddExactMatches(search.EntityType, storage.Alert_DEPLOYMENT.String()).WithPagination(pagination).ProtoQuery()
		_ = store.WalkByQuery(database.DBCtx, query, func(alert *storage.Alert) error {
			shouldAppend := false
			for _, container := range alert.GetDeployment().GetContainers() {
				newId := uuid.NewV5FromNonUUIDs(container.GetImage().GetName().GetFullName(), container.GetImage().GetId()).String()
				if container.GetImage().GetIdV2() != newId {
					container.GetImage().IdV2 = newId
					shouldAppend = true
				}
			}
			if shouldAppend {
				batch = append(batch, alert)
			}
			return nil
		})
		err := store.UpsertMany(database.DBCtx, batch)
		if err != nil {
			return err
		}
		if len(batch) < pageSize {
			break
		}
		page++
	}

	return nil
}
