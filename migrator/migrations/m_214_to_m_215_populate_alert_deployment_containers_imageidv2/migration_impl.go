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

func migrate(database *types.Databases) error {
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTableAlertsStmt)

	db := database.PostgresDB
	store := alertStore.New(db)

	pageSize := 10000
	for {
		batch := make([]*storage.Alert, 0, pageSize)
		pagination := search.NewPagination().Limit(int32(pageSize))
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
	}

	return nil
}
