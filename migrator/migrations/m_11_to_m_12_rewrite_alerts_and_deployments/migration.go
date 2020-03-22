package m11to12

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var migration = types.Migration{
	StartingSeqNum: 11,
	VersionAfter:   storage.Version{SeqNum: 12},
	Run:            func(_ *bolt.DB, _ *badger.DB) error { return nil },
}

func init() {
	migrations.MustRegisterMigration(migration)
}
