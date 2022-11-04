package m57tom58

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 57,
		VersionAfter:   &storage.Version{SeqNum: 58},
		Run: func(databases *types.Databases) error {
			err := updateRunSecretsVolumePolicy(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating 'Improper Usage of Orchestrator Secrets Volume' policy")
			}
			return nil
		},
	}

	policyBucketName  = []byte("policies")
	policyID          = "e971db42-e8d4-4a1d-a30c-41142ba54d71"
	oldPolicyCriteria = "VOLUME=/run/secrets"
	newPolicyCriteria = "VOLUME=(?:(?:[,\\[\\s]?)|(?:.*[,\\s]+))/run/secrets(?:$|[,\\]\\s]).*"
	policyFieldName   = "Dockerfile Line"
)

func updateRunSecretsVolumePolicy(db *bolt.DB) error {
	if exists, err := bolthelpers.BucketExists(db, policyBucketName); err != nil {
		return err
	} else if !exists {
		return nil
	}
	policyBucket := bolthelpers.TopLevelRef(db, policyBucketName)
	return policyBucket.Update(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policyID))
		if v == nil {
			return nil
		}

		var policy storage.Policy
		if err := proto.Unmarshal(v, &policy); err != nil {
			return errors.Wrapf(err, "unmarshaling migrated policy with id %q", policyID)
		}

		// Update the policy only if it has not already been altered by customer.
		if len(policy.GetPolicySections()) != 1 {
			return nil
		}

		section := policy.GetPolicySections()[0]
		if len(section.GetPolicyGroups()) != 1 {
			return nil
		}

		group := section.GetPolicyGroups()[0]
		if group == nil {
			return nil
		}

		if group.GetFieldName() != policyFieldName {
			return nil
		}

		if len(group.GetValues()) != 1 {
			return nil
		}

		value := group.GetValues()[0]
		if value == nil {
			return nil
		}

		// Check that the value actually matches the old version.
		if value.GetValue() != oldPolicyCriteria {
			return nil
		}

		// Update to the newer policy criteria
		value.Value = newPolicyCriteria

		policyBytes, err := proto.Marshal(&policy)
		if err != nil {
			return errors.Wrapf(err, "marshaling migrated policy %q with id %q", policy.GetName(), policy.GetId())
		}
		if err := b.Put([]byte(policyID), policyBytes); err != nil {
			return errors.Wrapf(err, "writing migrated policy with id %q to the store", policy.GetId())
		}
		return nil
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
