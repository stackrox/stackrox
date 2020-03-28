package m7to8

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	processIndicatorBucketName = []byte("process_indicators")
	deploymentBucketName       = []byte("deployments")
)

func upgradeAllProcessIndicators(db *bolt.DB) error {
	type clusterNSPair struct {
		clusterID string
		namespace string
	}
	var deployments = make(map[string]*clusterNSPair)
	deploymentBucket := bolthelpers.TopLevelRef(db, deploymentBucketName)
	err := deploymentBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			deployment := new(storage.Deployment)
			if err := proto.Unmarshal(v, deployment); err != nil {
				return err
			}
			deployments[deployment.GetId()] = &clusterNSPair{clusterID: deployment.GetClusterId(), namespace: deployment.GetNamespace()}
			return nil
		})
	})
	if err != nil {
		return err
	}
	processIndicatorBucket := bolthelpers.TopLevelRef(db, processIndicatorBucketName)
	err = processIndicatorBucket.Update(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			var indicator storage.ProcessIndicator
			if err := proto.Unmarshal(v, &indicator); err != nil {
				return errors.Wrap(err, "unmarshaling process indicator")
			}
			deployment := deployments[indicator.GetDeploymentId()]
			if deployment == nil {
				return nil
			}
			indicator.ClusterId = deployment.clusterID
			indicator.Namespace = deployment.namespace
			bytes, err := proto.Marshal(&indicator)
			if err != nil {
				return err
			}

			// It is not safe to add data to a bucket in a ForEach as the cursor will point to the wrong data but
			// according to https://github.com/boltdb/bolt/issues/268#issuecomment-66554702 it IS safe to update
			// existing data
			return b.Put(k, bytes)
		})
	})
	return err
}

func upgradeDataForSAC(db *bolt.DB, _ *badger.DB) error {
	if err := upgradeAllProcessIndicators(db); err != nil {
		return err
	}
	return nil
}

var (
	migration = types.Migration{
		StartingSeqNum: 7,
		VersionAfter:   storage.Version{SeqNum: 8},
		Run:            upgradeDataForSAC,
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
