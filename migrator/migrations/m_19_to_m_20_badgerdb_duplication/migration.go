package m19to20

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/badgermigration"
	"github.com/stackrox/rox/migrator/types"
)

var migration = types.Migration{
	StartingSeqNum: 19,
	VersionAfter:   storage.Version{SeqNum: 20},
	Run: func(databases *types.Databases) error {
		return badgermigration.RewriteData(databases.BoltDB, databases.BadgerDB)
	},
}

func init() {
	migrations.MustRegisterMigration(migration)
}
