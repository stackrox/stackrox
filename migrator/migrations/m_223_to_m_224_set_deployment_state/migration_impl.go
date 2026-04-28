package m223tom224

import (
	"context"

	"github.com/stackrox/rox/migrator/migrations/m_223_to_m_224_set_deployment_state/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	// Add deleted and state columns if they do not already exist.
	// No data backfill is needed because DEPLOYMENT_STATE_ACTIVE is the proto
	// zero value (0). Existing serialized protos without a state field
	// deserialize as ACTIVE, and the new integer column defaults to 0 in Go.
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableDeploymentsStmt)

	return nil
}
