package bolt

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	apiTokensBucket = []byte("apiTokens")
)

type storeImpl struct {
	*bolt.DB
}

// MustNew returns a ready-to-use store.
func MustNew(db *bolt.DB) store.Store {
	bolthelper.RegisterBucketOrPanic(db, apiTokensBucket)
	return &storeImpl{DB: db}
}

func (b *storeImpl) Upsert(token *storage.TokenMetadata) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "APIToken")

	if token.GetId() == "" {
		// This is most likely a programming error.
		return errors.New("token ID is empty")
	}

	bytes, err := proto.Marshal(token)
	if err != nil {
		return errors.Wrap(err, "proto marshaling")
	}

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		return bucket.Put([]byte(token.GetId()), bytes)
	})
}

func (b *storeImpl) Get(id string) (*storage.TokenMetadata, bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "APIToken")

	var token *storage.TokenMetadata
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		tokenBytes := bucket.Get([]byte(id))
		if tokenBytes == nil {
			return nil
		}
		token = new(storage.TokenMetadata)
		err := proto.Unmarshal(tokenBytes, token)
		if err != nil {
			return errors.Wrap(err, "proto unmarshaling")
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if token == nil {
		return nil, false, nil
	}
	return token, true, nil
}

func (b *storeImpl) Walk(fn func(*storage.TokenMetadata) error) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "APIToken")

	return b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(apiTokensBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var token storage.TokenMetadata
			err := proto.Unmarshal(v, &token)
			if err != nil {
				return errors.Wrap(err, "proto unmarshaling")
			}
			return fn(&token)
		})
	})
}
