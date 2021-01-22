package m54tom55

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
	oldVersion = "1"
	newVersion = "1.1"
)

var (
	migration = types.Migration{
		StartingSeqNum: 54,
		VersionAfter:   storage.Version{SeqNum: 55},
		Run: func(databases *types.Databases) error {
			err := migrateWhitelistsToExclusions(databases.BoltDB)
			if err != nil {
				return errors.Wrapf(err, "upgrading policies to version '%s", newVersion)
			}
			return nil
		},
	}

	policyBucket = []byte("policies")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

// migrateWhitelistsToExclusions assumes:
//   - all stored policies are of version "1";
//   - storage.Policy proto still defines the deprecated `.whitelists` field.
func migrateWhitelistsToExclusions(db *bolt.DB) error {
	numMigrated := 0

	err := db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}

		// First, enumerate all stored policies.
		var policyKeys [][]byte
		err := bucket.ForEach(func(key, _ []byte) error {
			policyKeys = append(policyKeys, key)
			return nil
		})

		// We can't proceed if we don't have a collection of all policy keys.
		if err != nil {
			return errors.Wrap(err, "failed to enumerate stored policies")
		}

		// Migrate and update policies one by one. Abort the transaction, and hence
		// the migration, in case of any error.
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

			migratePolicy(policy)

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

			numMigrated++
		}

		return nil
	})

	if err == nil {
		log.WriteToStderrf("successfully migrated %v policies to version %q", numMigrated, newVersion)
	}

	return err
}

// migratePolicy is semantically different from `EnsureConvertedToLatest()` in
// the booleanpolicy package in that it does not reject policies with both
// `exclusions` and `whitelists` but only logs the error. This is on purpose:
// we don't want to accept malformed policies as input but we tolerate this
// unlikely case here to avoid blocking a migration for a non-critical issue.
func migratePolicy(p *storage.Policy) {
	// It's fine to receive `exclusions` but not both `exclusions` and `whitelists`.
	if len(p.GetWhitelists()) > 0 && len(p.GetExclusions()) > 0 {
		log.WriteToStderrf("both 'exclusions' and 'whitelists' are present in policy with name %q; combining the fields", p.GetName())
	}

	p.Exclusions = append(p.Exclusions, p.Whitelists...)
	p.Whitelists = nil
	p.PolicyVersion = newVersion
}
