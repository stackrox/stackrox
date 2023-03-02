package {{.packageName}}

import (
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/migrator/migrations"
    "github.com/stackrox/rox/migrator/types"
)

const (
    startSeqNum = {{.startSequenceNumber}}
)

var (
    migration = types.Migration{
        StartingSeqNum: startSeqNum,
        VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
        Run: func(database *types.Databases) error {
            // Migration code comes here
            return nil
        },
    }
)

func init() {
    migrations.MustRegisterMigration(migration)
}

// Additional code to support the migration
