package {{.packageName}}

import (
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/migrator/migrations"
    "github.com/stackrox/rox/migrator/types"
)
// TODO: generate/write and import store code required for the migration.

const (
    startSeqNum = {{.startSequenceNumber}}
)

var (
    migration = types.Migration{
        StartingSeqNum: startSeqNum,
        VersionAfter:   &storage.Version{SeqNum: int32(startSeqNum + 1)},
        Run: func(database *types.Databases) error {
            // TODO: Migration code comes here
            return nil
        },
    }
)

func init() {
    migrations.MustRegisterMigration(migration)
}

// TODO: Write the additional code to support the migration

// TODO: remove any pending TODO
