package store

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	configBucket = []byte("config")
	configKey    = []byte("\x00")
)

// Store provides storage functionality for the Central config.
//go:generate mockgen-wrapper Store
type Store interface {
	GetConfig() (*storage.Config, error)
	UpdateConfig(*storage.Config) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, configBucket)
	return &storeImpl{
		DB: db,
	}
}

type storeImpl struct {
	*bolt.DB
}

// GetConfig returns Central's config
func (b *storeImpl) GetConfig() (*storage.Config, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Config")
	var config storage.Config
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(configBucket)
		val := bucket.Get(configKey)
		if val == nil {
			return nil
		}
		return proto.Unmarshal(val, &config)
	})
	return &config, err
}

// UpdateConfig updates Central config
func (b *storeImpl) UpdateConfig(config *storage.Config) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Config")
	data, err := proto.Marshal(config)
	if err != nil {
		return err
	}
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(configBucket)
		return bucket.Put(configKey, data)
	})
}
