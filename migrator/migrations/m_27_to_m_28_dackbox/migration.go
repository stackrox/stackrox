package m27tom28

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 27,
		VersionAfter:   storage.Version{SeqNum: 28},
		Run: func(_ *types.Databases) error {
			// Migration was aborted due to a missed release.
			// Migration was moved to m_32_to_m_33_dackbox
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
