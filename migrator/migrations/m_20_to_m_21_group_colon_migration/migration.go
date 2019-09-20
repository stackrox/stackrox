package m20to21

import (
	"bytes"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var groupsBucket = []byte("groups")

var migration = types.Migration{
	StartingSeqNum: 20,
	VersionAfter:   storage.Version{SeqNum: 21},
	Run: func(db *bolt.DB, badgerDB *badger.DB) error {
		return rewrite(db)
	},
}

func rewrite(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(groupsBucket)
		return bucket.ForEach(func(k, v []byte) error {
			newKey := bytes.ReplaceAll(k, []byte(":"), []byte("\x00"))
			if err := bucket.Put(newKey, v); err != nil {
				return err
			}
			return bucket.Delete(k)
		})
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
