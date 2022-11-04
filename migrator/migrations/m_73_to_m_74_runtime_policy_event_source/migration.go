package m73tom74

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 73,
		VersionAfter:   &storage.Version{SeqNum: 74},
		Run: func(databases *types.Databases) error {
			err := migrateDefaultRuntimeEventSource(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "upgrading runtime policies to deployment event source")
			}
			return nil
		},
	}

	policyBucket = []byte("policies")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateDefaultRuntimeEventSource(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}

		// Migrate and update policies one by one. Abort the transaction, and hence
		// the migration, in case of any error.
		// First, enumerate all stored policies.
		var policyKeys [][]byte
		err := bucket.ForEach(func(key, obj []byte) error {
			policy := &storage.Policy{}
			if err := proto.Unmarshal(obj, policy); err != nil {
				// Unclear how to recover from unmarshal error, abort the transaction.
				return errors.Wrapf(err, "failed to unmarshal policy data for key %q", key)
			}
			if appliesAtRunTime(policy) {
				policyKeys = append(policyKeys, key)
			}
			return nil
		})
		// We can't proceed if we don't have a collection of all policy keys.
		if err != nil {
			return errors.Wrap(err, "failed to enumerate stored policies")
		}

		for _, key := range policyKeys {
			obj := bucket.Get(key)
			if obj == nil {
				// This is unexpected, abort the transaction.
				return errors.Errorf("expected policy with key %q not found", key)
			}
			policy := &storage.Policy{}
			if err := proto.Unmarshal(obj, policy); err != nil {
				// Unclear how to recover from unmarshal error, abort the transaction.
				return errors.Wrapf(err, "failed to unmarshal policy data for key %q", key)
			}

			policy.EventSource = storage.EventSource_DEPLOYMENT_EVENT
			obj, err := proto.Marshal(policy)
			if err != nil {
				// Unclear how to recover from marshal error, abort the transaction.
				return errors.Wrapf(err, "failed to marshal migrated policy %q for key %q", policy.GetName(), policy.GetId())
			}

			// Update successfully migrated policy. No need to update secondary
			// mappings because neither policy name nor id has changed.
			if err := bucket.Put(key, obj); err != nil {
				// Unclear how to recover if we cannot update the record.
				return errors.Wrapf(err, "failed to write migrated policy with key %q to the store", key)
			}
		}
		return nil
	})
}

func appliesAtRunTime(policy *storage.Policy) bool {
	for _, stage := range policy.GetLifecycleStages() {
		if stage == storage.LifecycleStage_RUNTIME {
			return true
		}
	}
	return false
}
