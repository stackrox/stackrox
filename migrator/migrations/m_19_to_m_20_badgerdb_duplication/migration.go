package m19to20

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/badgermigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/features"
)

var migration = types.Migration{
	StartingSeqNum: 19,
	VersionAfter:   storage.Version{SeqNum: 20},
	Run: func(db *bolt.DB, badgerDB *badger.DB) error {
		if !features.BadgerDB.Enabled() {
			return nil
		}
		return badgermigration.RewriteData(db, badgerDB)
	},
}

func init() {
	migrations.MustRegisterMigration(migration)
}
