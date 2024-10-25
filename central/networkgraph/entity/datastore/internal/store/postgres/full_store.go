package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/rox/pkg/env"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
)

const (
	networkFlowsTable    = pkgSchema.NetworkFlowsTableName
	networkEntitiesTable = pkgSchema.NetworkEntitiesTableName

	pruneOrphanExternalNetworkEntitiesSrcStmt = `DELETE FROM %s entity WHERE NOT EXISTS
		(SELECT 1 FROM %s flow WHERE flow.Props_SrcEntity_Type = 4
		AND flow.Props_SrcEntity_Id = entity.Info_Id
		AND entity.Info_ExternalSource_Learned = true);`

	pruneOrphanExternalNetworkEntitiesDstStmt = `DELETE FROM %s entity WHERE NOT EXISTS
		(SELECT 1 FROM %s flow WHERE flow.Props_DstEntity_Type = 4
		AND flow.Props_DstEntity_Id = entity.Info_Id
		AND entity.Info_ExternalSource_Learned = true);`
)

var (
	queryTimeout = env.PostgresDefaultNetworkFlowQueryTimeout.DurationSetting()
)

// NewFullStore augments the generated store with RemoveOrphanedEntities function.
func NewFullStore(db postgres.DB) store.EntityStore {
	return &fullStoreImpl{
		db:    db,
		Store: New(db),
	}
}

type fullStoreImpl struct {
	db postgres.DB

	Store
}

// RemoveOrphanedEntities prunes 'discovered' external entities that are not referenced by any flow.
func (f *fullStoreImpl) RemoveOrphanedEntities(ctx context.Context) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "NetworkEntitiesPruning")

	pruneStmt := fmt.Sprintf(pruneOrphanExternalNetworkEntitiesSrcStmt, networkEntitiesTable, networkFlowsTable)
	err := f.pruneEntities(ctx, pruneStmt)
	if err != nil {
		return err
	}

	pruneStmt = fmt.Sprintf(pruneOrphanExternalNetworkEntitiesDstStmt, networkEntitiesTable, networkFlowsTable)
	return f.pruneEntities(ctx, pruneStmt)
}

func (f *fullStoreImpl) pruneEntities(ctx context.Context, deleteStmt string) error {
	conn, err := f.db.Acquire(ctx)
	if err != nil {
		return nil
	}

	defer conn.Release()

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if _, err := conn.Exec(ctx, deleteStmt); err != nil {
		return err
	}

	return nil
}
