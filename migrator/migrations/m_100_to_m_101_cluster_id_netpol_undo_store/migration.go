package m100tom101

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 100,
		VersionAfter:   &storage.Version{SeqNum: 101},
		Run: func(databases *types.Databases) error {
			if err := addClusterIDToNetworkPolicyApplicationUndoRecord(databases.BoltDB); err != nil {
				return errors.Wrap(err,
					"updating undo store with cluster id field")
			}
			return nil
		},
	}
	bucket = []byte("networkpolicies-undo")
)

func addClusterIDToNetworkPolicyApplicationUndoRecord(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucket)
		return bucket.ForEach(func(k, v []byte) error {
			var np storage.NetworkPolicyApplicationUndoRecord
			if err := proto.Unmarshal(v, &np); err != nil {
				return err
			}
			np.ClusterId = string(k)
			data, err := proto.Marshal(&np)
			if err != nil {
				return err
			}
			return bucket.Put(k, data)
		})
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
