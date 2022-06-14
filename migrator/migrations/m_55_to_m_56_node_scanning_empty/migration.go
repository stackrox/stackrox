package m55tom56

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/migrator/migrations"
	"github.com/stackrox/stackrox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 55,
		VersionAfter:   storage.Version{SeqNum: 56},
		Run: func(databases *types.Databases) error {
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
