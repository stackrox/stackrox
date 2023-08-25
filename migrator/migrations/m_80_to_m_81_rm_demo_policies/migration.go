package m80tom81

import (
	"embed"

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
		StartingSeqNum: 80,
		VersionAfter:   &storage.Version{SeqNum: 81},
		Run: func(databases *types.Databases) error {
			err := rmDemoPolicies(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "deprecating default system policies")
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

func rmDemoPolicies(db *bolt.DB) error {
	policiesToRm, err := policymigrationhelper.ReadPolicyFromDir(policiesFS, "policies")
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}
		for _, policyToRm := range policiesToRm {
			val := bucket.Get([]byte(policyToRm.GetId()))
			if val == nil {
				log.WriteToStderrf("default system policy with ID %s not found in DB. Continuing", policyToRm.GetId())
				continue
			}

			storedPolicy := &storage.Policy{}
			if err := proto.Unmarshal(val, storedPolicy); err != nil {
				return errors.Wrapf(err, "unmarshaling policy with ID %q", policyToRm.GetId())
			}

			if !proto.Equal(storedPolicy, policyToRm) {
				return nil
			}

			if err := bucket.Delete([]byte(policyToRm.GetId())); err != nil {
				return errors.Wrapf(err, "removing policy %s", policyToRm.GetId())
			}
		}
		return nil
	})
}
