package m6to7

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	imageBucketName = []byte("imageBucket")
)

func readAndWriteImages(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(imageBucketName)
		return bucket.ForEach(func(k, v []byte) error {
			var image storage.Image
			if err := proto.Unmarshal(v, &image); err != nil {
				return err
			}
			image.LastUpdated = protoTypes.TimestampNow()

			data, err := proto.Marshal(&image)
			if err != nil {
				return err
			}
			return bucket.Put(k, data)
		})
	})

}

var (
	migration = types.Migration{
		StartingSeqNum: 6,
		VersionAfter:   storage.Version{SeqNum: 7},
		Run: func(db *bolt.DB, _ *badger.DB) error {
			return readAndWriteImages(db)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}
