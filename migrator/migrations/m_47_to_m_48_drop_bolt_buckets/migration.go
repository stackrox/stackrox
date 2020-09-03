package m47tom48

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 47,
		VersionAfter:   storage.Version{SeqNum: 48},
		Run:            dropBoltBuckets,
	}

	bucketsToBeDropped = []string{
		"risk",
		"processWhitelists2",
		"service_accounts",
		"k8sroles",
		"rolebindings",
		"secrets",
		"secrets_list",
		"namespaces",
		"processWhitelistResults",

		"clusters",
		"cluster_status",
		"clusters_last_contact",
		"apiTokens",

		"compliance-run-results",
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func dropBoltBuckets(databases *types.Databases) error {
	for _, bucket := range bucketsToBeDropped {
		err := databases.BoltDB.Update(func(tx *bbolt.Tx) error {
			bucketKey := []byte(bucket)
			if tx.Bucket(bucketKey) == nil {
				return nil
			}
			return tx.DeleteBucket(bucketKey)
		})
		if err != nil {
			return errors.Wrapf(err, "dropping bucket %q", bucket)
		}
	}
	return nil
}
