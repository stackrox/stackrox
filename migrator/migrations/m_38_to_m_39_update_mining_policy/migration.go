package m38tom39

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	policyBucketName  = []byte("policies")
	policyID          = "e9635b83-4ec5-4e7a-9be1-1bcdd6d82bb7"
	oldPolicyCriteria = ".*sgminer|.*cgminer|.*cpuminer|.*minerd|.*geth|.*ethminer|.*xmr-stak-cpu|.*xmr-stak-amd|.*xmr-stak-nvidia|.*xmrminer|.*cpuminer-multi"
	newPolicyCriteria = ".*sgminer|.*cgminer|.*cpuminer|.*minerd|.*geth|.*ethminer|.*xmr-stak.*|.*xmrminer|.*cpuminer-multi"
	processName       = "Process Name"
)

func updateCryptoMiningPolicyCriteria(db *bolt.DB) error {
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

		for _, section := range policy.GetPolicySections() {
			for _, group := range section.GetPolicyGroups() {
				if group.GetFieldName() != processName {
					continue
				}

				for _, value := range group.GetValues() {
					if value.GetValue() == oldPolicyCriteria {
						value.Value = newPolicyCriteria
						break
					}
				}
			}
		}

		policyBytes, err := proto.Marshal(&policy)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(policyID), policyBytes); err != nil {
			return errors.Wrap(err, "failed to insert")
		}
		return nil
	})
}

var (
	migration = types.Migration{
		StartingSeqNum: 38,
		VersionAfter:   storage.Version{SeqNum: 39},
		Run: func(databases *types.Databases) error {
			err := updateCryptoMiningPolicyCriteria(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating Cryptocurrency Mining Process Execution policy")
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
