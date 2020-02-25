package m27tom28

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 27,
		VersionAfter:   storage.Version{SeqNum: 28},
		Run: func(_ *bolt.DB, db *badger.DB) error {
			err := migrateDeploymentsAndImages(db)
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
