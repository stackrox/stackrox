package m172tom173

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_network_flow_indexes_to_hash/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"gorm.io/gorm"
)

var (
	startSeqNum = 172

	migration = types.Migration{
		StartingSeqNum: startSeqNum,
		VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)}, // 173
		Run: func(databases *types.Databases) error {
			err := UpdateIndexesToHash(databases.GormDB, databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "updating policy categories schema")
			}
			return nil
		},
	}

	log = loghelper.LogWrapper{}
)

// UpdateIndexesToHash updates the btree network flow indexes to be hash
func UpdateIndexesToHash(gormDB *gorm.DB, db *postgres.DB) error {
	log.WriteToStderr("SHREWS -- about to remove some indexes")
	// Automigrate does not remove or update indexes, it only creates them.
	// Remove index
	gormDB.Migrator().DropIndex(&schema.NetworkFlows{}, "network_flows_cluster")
	gormDB.Migrator().DropIndex(&schema.NetworkFlows{}, "network_flows_dst")
	gormDB.Migrator().DropIndex(&schema.NetworkFlows{}, "network_flows_src")

	// Now apply the updated schema to get the updated indexes
	pgutils.CreateTableFromModel(context.Background(), gormDB, frozenSchema.CreateTableNetworkFlowsStmt)
	log.WriteToStderr("SHREWS -- network flow updated????")

	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
