package m36tom37

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

const (
	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: 36,
		VersionAfter:   storage.Version{SeqNum: 37},
		Run: func(databases *types.Databases) error {
			return migrate(databases)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrate(databases *types.Databases) error {
	if err := migrateBadger(databases); err != nil {
		return errors.Wrap(err, "migrating badger -> rocksdb")
	}
	if err := migrateBolt(databases); err != nil {
		return errors.Wrap(err, "migrating bolt -> rocksdb")
	}
	return nil
}
