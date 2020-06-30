package m32tom33

import (
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 32,
		VersionAfter:   storage.Version{SeqNum: 33},
		Run: func(databases *types.Databases) error {
			err := migrateDeploymentsAndImages(databases.BadgerDB)
			if err != nil {
				return errors.Wrap(err, "updating images and deployments to dackbox")
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateDeploymentsAndImages(db *badger.DB) error {
	if err := rewriteDeployments(db); err != nil {
		return err
	}
	return rewriteImages(db)
}
