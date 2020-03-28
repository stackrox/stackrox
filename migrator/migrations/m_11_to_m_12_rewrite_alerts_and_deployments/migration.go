package m11to12

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var migration = types.Migration{
	StartingSeqNum: 11,
	VersionAfter:   storage.Version{SeqNum: 12},
	Run:            func(_ *bolt.DB, _ *badger.DB) error { return nil },
}

func init() {
	migrations.MustRegisterMigration(migration)
}
