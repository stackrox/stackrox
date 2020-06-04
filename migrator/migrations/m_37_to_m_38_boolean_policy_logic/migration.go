package m37tom38

import (
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/features"
)

var (
	migration = types.Migration{
		StartingSeqNum: 37,
		VersionAfter:   storage.Version{SeqNum: 38},
		Run: func(databases *types.Databases) error {
			if !features.BooleanPolicyLogic.Enabled() {
				return nil
			}
			err := migrateLegacyPoliciesToBPL(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "upgrading legacy policies to boolean policies")
			}
			return nil
		},
	}

	policyBucket           = []byte("policies")
	uniqueBucket           = []byte("policies-unique")
	mapperBucket           = []byte("policies-mapper")
	unmigratableBucketName = []byte("unmigratablePolicies37To38")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateLegacyPoliciesToBPL(db *bbolt.DB) error {
	var policiesToMigrate []*storage.Policy // Should be able to hold all policies in memory easily
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			policy := &storage.Policy{}
			if err := proto.Unmarshal(v, policy); err != nil {
				// If anything fails to unmarshal roll back the transaction and abort
				return errors.Wrapf(err, "Failed to unmarshal policy data for key %s", k)
			}
			if policy.GetPolicyVersion() == version {
				return nil // already migrated
			}
			policiesToMigrate = append(policiesToMigrate, policy)
			return nil
		})
	})

	if err != nil {
		return errors.Wrap(err, "reading policy data")
	}

	if len(policiesToMigrate) == 0 {
		return nil // nothing to do
	}

	migratedPolicies := make([]*storage.Policy, 0, len(policiesToMigrate))
	var unmigratablePolicies []*storage.Policy
	for _, unmigratedPolicy := range policiesToMigrate {
		migratedPolicy, err := CloneAndEnsureConverted(unmigratedPolicy)
		if err != nil {
			unmigratablePolicies = append(unmigratablePolicies, unmigratedPolicy)
			log.WriteToStderrf("failed migrate policy %s:%s: %v it has been removed and stored in the %s bucket", unmigratedPolicy.GetName(), unmigratedPolicy.GetId(), err, unmigratableBucketName)
			continue
		}
		migratedPolicy.Fields = unmigratedPolicy.Fields.Clone()
		migratedPolicies = append(migratedPolicies, migratedPolicy)
	}

	if len(unmigratablePolicies) > 0 {
		if err := registerBucket(db, unmigratableBucketName); err != nil {
			return err
		}
	}

	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %s not found", policyBucket)
		}

		// Store successfully migrated policies.  We don't need to change the name/ID cross index.
		for _, policy := range migratedPolicies {
			if err := storePolicy(policy, bucket); err != nil {
				return err
			}
		}

		if len(unmigratablePolicies) == 0 {
			return nil
		}

		// If any policies couldn't be migrated remove them from the policy bucket and the name/ID cross index and add
		// the policy to a separate bucket for later inspection.
		if _, err := tx.CreateBucketIfNotExists(unmigratableBucketName); err != nil {
			return errors.Wrap(err, "creating unmigratable bucket")
		}

		uBucket := tx.Bucket(uniqueBucket)
		if uBucket == nil {
			return errors.Errorf("bucket %s not found", uniqueBucket)
		}
		mBucket := tx.Bucket(mapperBucket)
		if mBucket == nil {
			return errors.Errorf("bucket %s not found", mapperBucket)
		}
		unmigratableBucket := tx.Bucket(unmigratableBucketName)
		if unmigratableBucket == nil {
			return errors.Errorf("bucket %s not found", unmigratableBucketName)
		}
		for _, policy := range unmigratablePolicies {
			if err := bucket.Delete([]byte(policy.GetId())); err != nil {
				return errors.Wrapf(err, "failed to delete policy %s:%s", policy.GetName(), policy.GetId())
			}
			if err := uBucket.Delete([]byte(policy.GetName())); err != nil {
				return errors.Wrapf(err, "failed to delete policy from name bucket %s:%s", policy.GetName(), policy.GetId())
			}
			if err := mBucket.Delete([]byte(policy.GetId())); err != nil {
				return errors.Wrapf(err, "failed to delete policy from ID bucket %s:%s", policy.GetName(), policy.GetId())
			}
			if err := storePolicy(policy, unmigratableBucket); err != nil {
				return err
			}
		}

		return nil
	})
}

func registerBucket(db *bbolt.DB, bucket []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return errors.Wrap(err, "create bucket")
		}
		return nil
	})
}

func storePolicy(policy *storage.Policy, bucket *bbolt.Bucket) error {
	bytes, err := proto.Marshal(policy)
	if err != nil {
		// If anything fails to marshal roll back the transaction and abort
		return errors.Wrapf(err, "failed to marshal migrated policy %s:%s", policy.GetName(), policy.GetId())
	}
	// No need to update secondary mappings, we haven't changed the name and the name mapping just references the ID.
	if err := bucket.Put([]byte(policy.GetId()), bytes); err != nil {
		return err
	}
	return nil
}
