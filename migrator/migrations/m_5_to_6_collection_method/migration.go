package m5to6

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	clusterBucketName = []byte("clusters")
)

func updateRuntimeSupportToCollectionMethod(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterBucketName)
		return bucket.ForEach(func(k, v []byte) error {
			var cluster storage.Cluster
			if err := proto.Unmarshal(v, &cluster); err != nil {
				return err
			}
			// If deprecated runtime support is not set then no need to rewrite
			if !cluster.GetRuntimeSupport() {
				return nil
			}
			cluster.CollectionMethod = storage.CollectionMethod_KERNEL_MODULE

			data, err := proto.Marshal(&cluster)
			if err != nil {
				return err
			}
			return bucket.Put(k, data)
		})
	})
}

var (
	migration = types.Migration{
		StartingSeqNum: 5,
		VersionAfter:   storage.Version{SeqNum: 6},
		Run: func(db *bolt.DB, _ *badger.DB) error {
			return updateRuntimeSupportToCollectionMethod(db)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
