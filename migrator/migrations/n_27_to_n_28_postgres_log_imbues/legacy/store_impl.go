// This file was originally generated with
// //go:generate cp ../../../../central/logimbue/store/bolt/store_impl.go .

package legacy

import (
	"context"
	"encoding/binary"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/uuid"
	bolt "go.etcd.io/bbolt"
)

var logsBucket = []byte("logs")

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) *storeImpl {
	bolthelper.RegisterBucketOrPanic(db, logsBucket)
	return &storeImpl{
		DB: db,
	}
}

type storeImpl struct {
	*bolt.DB
}

// GetAll returns all of the logs stored in the DB.
func (b *storeImpl) GetAll(_ context.Context) ([]*storage.LogImbue, error) {
	var logs []*storage.LogImbue
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(logsBucket).Cursor()

		for k, v := bucket.First(); k != nil; k, v = bucket.Next() {
			seconds := binary.LittleEndian.Uint64(k)

			log := make([]byte, len(v))
			copy(log, v)

			logs = append(logs, &storage.LogImbue{
				Id: uuid.NewV4().String(),
				Timestamp: &types.Timestamp{
					Seconds: int64(seconds),
				},
				Log: log,
			})
		}
		return nil
	})
	return logs, err
}

// Upsert adds a log to bolt.
func (b *storeImpl) Upsert(_ context.Context, log *storage.LogImbue) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(logsBucket)

		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(log.GetTimestamp().GetSeconds()))

		return bucket.Put(b, log.GetLog())
	})
}
