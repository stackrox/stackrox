package m59tom60

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

const (
	dockerCIS = "Docker CIS"
)

var (
	migration = types.Migration{
		StartingSeqNum: 59,
		VersionAfter:   &storage.Version{SeqNum: 60},
		Run: func(databases *types.Databases) error {
			return migrateNewPolicyCategories(databases.BoltDB)
		},
	}

	policyBucket = []byte("policies")

	// This is a list of policies to which we should add the Docker CIS category
	policiesToUpdate = []string{
		"80267b36-2182-4fb3-8b53-e80c031f4ad8",
		"886c3c94-3a6a-4f2b-82fc-d6bf5a310840",
		"fe9de18b-86db-44d5-a7c4-74173ccffe2e",
		"618e65ca-737b-4fec-bb42-57a04e7dfc28",
		"8ac93556-4ad4-4220-a275-3f518db0ceb9",
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateNewPolicyCategories(db *bolt.DB) error {
	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}

		for _, migrateID := range policiesToUpdate {
			policyBytes := bucket.Get([]byte(migrateID))
			if policyBytes == nil {
				log.WriteToStderrf("no policy exists for ID %s in policy category migration.  Continuing", migrateID)
				continue
			}

			policy := &storage.Policy{}
			if err := proto.Unmarshal(policyBytes, policy); err != nil {
				// Unclear how to recover from unmarshal error, abort the transaction.
				return errors.Wrapf(err, "failed to unmarshal policy data for key %q", migrateID)
			}
			migratePolicy(policy)

			obj, err := proto.Marshal(policy)
			if err != nil {
				// Unclear how to recover from marshal error, abort the transaction.
				return errors.Wrapf(err, "failed to marshal migrated policy %q for key %q", policy.GetName(), policy.GetId())
			}

			// Update successfully migrated policy. No need to update secondary
			// mappings because neither policy name nor id has changed.
			if err := bucket.Put([]byte(migrateID), obj); err != nil {
				// Unclear how to recover if we cannot update the record.
				return errors.Wrapf(err, "failed to write migrated policy with key %q to the store", migrateID)
			}
		}

		return nil
	})

	if err == nil {
		log.WriteToStderrf("successfully migrated Docker CIS policy categories")
	}

	return err
}

func migratePolicy(p *storage.Policy) {
	for _, existingCategory := range p.GetCategories() {
		if existingCategory == dockerCIS {
			return
		}
	}

	p.Categories = append(p.Categories, dockerCIS)
}
