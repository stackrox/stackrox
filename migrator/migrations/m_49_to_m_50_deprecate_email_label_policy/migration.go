package m49tom50

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
	policyBucketName = []byte("policies")
	policyID         = "60e7c7f3-dc78-4367-9e9a-68aa3b7467f0"
	policyCriteria   = "email=[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\\.[a-zA-Z0-9-.]+"
	policyFieldName  = "Required Label"
)

func deprecateRequiredLabelEmailPolicy(db *bolt.DB) error {
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
			return err
		}

		// Delete the policy only if it has not been altered.
		if len(policy.GetPolicySections()) != 1 {
			return nil
		}

		section := policy.GetPolicySections()[0]
		if len(section.GetPolicyGroups()) != 1 {
			return nil
		}

		group := section.GetPolicyGroups()[0]
		if group.GetFieldName() != policyFieldName {
			return nil
		}

		if len(group.GetValues()) != 1 {
			return nil
		}

		value := group.GetValues()[0]
		if value.GetValue() != policyCriteria {
			return nil
		}

		if err := b.Delete([]byte(policyID)); err != nil {
			return errors.Wrap(err, "failed to delete")
		}
		return nil
	})
}

var (
	migration = types.Migration{
		StartingSeqNum: 49,
		VersionAfter:   storage.Version{SeqNum: 50},
		Run: func(databases *types.Databases) error {
			err := deprecateRequiredLabelEmailPolicy(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "deprecating 'Required Label: Email' policy")
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
