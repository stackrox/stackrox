package m0tom1

import (
	"github.com/dgraph-io/badger"
	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	versionBucket = []byte("version")

	// This is the initial migration, which takes us from a pre-2.4 world to a 2.4 world.
	// Note that this migration is special, in the sense that you CANNOT make any strong assumptions
	// about what the DB will look like, since this can be called from any version starting from 2.3.11 to 2.13.15.
	migration23To24 = types.Migration{
		StartingSeqNum: 0,
		Run: func(db *bbolt.DB, _ *badger.DB) error {
			return db.Update(func(tx *bbolt.Tx) error {
				_, err := tx.CreateBucketIfNotExists(versionBucket)
				return err
			})
		},
		VersionAfter: storage.Version{SeqNum: 1},
	}
)

func init() {
	migrations.MustRegisterMigration(migration23To24)
}
