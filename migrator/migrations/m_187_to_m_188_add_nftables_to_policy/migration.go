package m187tom188

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	migration = types.Migration{
		StartingSeqNum: 187,
		VersionAfter:   &storage.Version{SeqNum: 188},
		Run: func(databases *types.Databases) error {
			err := updatePolicies(databases.PostgresDB)
			if err != nil {
				return errors.Wrap(err, "updating policies")
			}
			return nil
		},
	}
)

func updatePolicies(_ postgres.DB) error {
	// OBE for testing
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
