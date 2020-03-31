package m18to19

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/badgermigration"
	"github.com/stackrox/rox/migrator/types"
)

var migration = types.Migration{
	StartingSeqNum: 18,
	VersionAfter:   storage.Version{SeqNum: 19},
	Run: func(databases *types.Databases) error {
		return badgermigration.RewriteData(databases.BoltDB, databases.BadgerDB)
	},
}

func init() {
	migrations.MustRegisterMigration(migration)
}
