package m55tom56

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 55,
		VersionAfter:   &storage.Version{SeqNum: 56},
		Run: func(databases *types.Databases) error {
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
