package m19to20

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/badgermigration"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var migration = types.Migration{
	StartingSeqNum: 19,
	VersionAfter:   storage.Version{SeqNum: 20},
	Run: func(db *bolt.DB, badgerDB *badger.DB) error {
		return badgermigration.RewriteData(db, badgerDB)
	},
}

func init() {
	migrations.MustRegisterMigration(migration)
}
