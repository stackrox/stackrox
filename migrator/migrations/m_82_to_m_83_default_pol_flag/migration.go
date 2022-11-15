package m82tom83

import (
	"embed"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 82,
		VersionAfter:   &storage.Version{SeqNum: 83},
		Run: func(databases *types.Databases) error {
			err := updatePoliciesWithDefaultFlag(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating default system policies with default flag")
			}
			return nil
		},
	}

	policyBucket = []byte("policies")

	//go:embed policies/*.json
	policiesFS embed.FS
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func updatePoliciesWithDefaultFlag(db *bolt.DB) error {
	slimDefaultPolicies, err := policymigrationhelper.ReadPolicyFromDir(policiesFS, "policies")
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}

		for _, policy := range slimDefaultPolicies {
			val := bucket.Get([]byte(policy.GetId()))
			if val == nil {
				log.WriteToStderrf("default system policy with ID %s not found in DB. Continuing", policy.GetId())
				continue
			}

			storedPolicy := &storage.Policy{}
			if err := proto.Unmarshal(val, storedPolicy); err != nil {
				return errors.Wrapf(err, "unmarshaling stored policy with id %q", policy.GetId())
			}

			// Mark the policies as default only if the criteria is locked. If the criteria is not locked, it can be
			// changed in future. We do not want to prohibit users from deleting the policies that may have been
			// changed or can change in future. All new installation or new policies starting 65.0 have policy criteria
			// locked. Therefore, by checking this we ensure that we not migrate the policies prior to 65.0.
			if !storedPolicy.GetCriteriaLocked() {
				continue
			}

			// As long as the policy criteria is same as the one we shipped, we can mark the policy as default policy,
			// otherwise not.
			if !reflect.DeepEqual(storedPolicy.GetPolicySections(), policy.GetPolicySections()) {
				continue
			}

			storedPolicy.IsDefault = true

			data, err := proto.Marshal(storedPolicy)
			if err != nil {
				return errors.Wrapf(err, "marshalling policy %s", policy.GetId())
			}

			if err := bucket.Put([]byte(policy.GetId()), data); err != nil {
				return errors.Wrapf(err, "adding policy %s", policy.GetId())
			}
		}
		return nil
	})
}
