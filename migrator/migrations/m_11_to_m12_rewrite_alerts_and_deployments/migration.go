package m11to12

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var migration = types.Migration{
	StartingSeqNum: 11,
	VersionAfter:   storage.Version{SeqNum: 12},
	Run:            rewriteData,
}

var (
	deploymentBucket = []byte("deployments")
	alertBucket      = []byte("alerts")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func rewriteData(db *bolt.DB, _ *badger.DB) error {
	if err := rewriteAlerts(db); err != nil {
		return err
	}
	return rewriteDeployments(db)
}

func rewriteResource(bucket *bolt.Bucket, k, v []byte, msg proto.Message) error {
	if err := proto.Unmarshal(v, msg); err != nil {
		return err
	}
	newValue, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return bucket.Put(k, newValue)
}

func rewriteBucket(db *bolt.DB, bucketName []byte, msg proto.Message) error {
	var (
		nextKey  []byte
		finished bool
	)

	batchSize := 10000
	for !finished {
		err := db.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(bucketName)
			cursor := bucket.Cursor()
			k, v := cursor.Seek(nextKey)
			// empty DB will have nil key here
			if k == nil {
				finished = true
				return nil
			}
			for i := 0; i < batchSize; i++ {
				if err := rewriteResource(bucket, k, v, msg); err != nil {
					return err
				}
				msg.Reset()
				if k, v = cursor.Next(); k == nil {
					finished = true
					return nil
				}
			}
			nextKey = k
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func rewriteAlerts(db *bolt.DB) error {
	var alert storage.Alert
	return rewriteBucket(db, alertBucket, &alert)
}

func rewriteDeployments(db *bolt.DB) error {
	var deployment storage.Deployment
	return rewriteBucket(db, deploymentBucket, &deployment)
}
