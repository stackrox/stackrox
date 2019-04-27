package store

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) getWhitelist(tx *bolt.Tx, id string) (*storage.ProcessWhitelist, error) {
	bucket := tx.Bucket(processWhitelistBucket)
	val := bucket.Get([]byte(id))
	if val == nil {
		return nil, nil
	}

	whitelist := new(storage.ProcessWhitelist)
	err := proto.Unmarshal(val, whitelist)
	if err != nil {
		return nil, err
	}
	return whitelist, nil
}

func (b *storeImpl) GetWhitelist(id string) (*storage.ProcessWhitelist, error) {
	var whitelist *storage.ProcessWhitelist
	err := b.View(func(tx *bolt.Tx) error {
		var err error
		whitelist, err = b.getWhitelist(tx, id)
		return err
	})
	return whitelist, err
}

func (b *storeImpl) GetWhitelists() ([]*storage.ProcessWhitelist, error) {
	var whitelists []*storage.ProcessWhitelist
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(processWhitelistBucket)
		whitelists = make([]*storage.ProcessWhitelist, 0, b.Stats().KeyN)
		return b.ForEach(func(k, v []byte) error {
			var whitelist storage.ProcessWhitelist
			if err := proto.Unmarshal(v, &whitelist); err != nil {
				return err
			}
			whitelists = append(whitelists, &whitelist)
			return nil
		})
	})
	return whitelists, err
}

func storeWhitelist(id []byte, whitelist *storage.ProcessWhitelist, bucket *bolt.Bucket) error {
	whitelist.Id = string(id)
	bytes, err := proto.Marshal(whitelist)
	if err != nil {
		return err
	}
	return bucket.Put(id, bytes)
}

func addWhitelist(id []byte, whitelist *storage.ProcessWhitelist, tx *bolt.Tx) error {
	bucket := tx.Bucket(processWhitelistBucket)
	if exists := bolthelper.ExistsBytes(bucket, id); exists {
		return errors.New(fmt.Sprintf("whitelist %s already exists", string(id)))
	}
	whitelist.Created = types.TimestampNow()
	whitelist.LastUpdate = whitelist.GetCreated()
	genDuration := env.WhitelistGenerationDuration.DurationSetting()
	lockTimestamp, err := types.TimestampProto(time.Now().Add(genDuration))
	if err == nil {
		whitelist.StackRoxLockedTimestamp = lockTimestamp
	}
	return storeWhitelist(id, whitelist, bucket)
}

func (b *storeImpl) AddWhitelist(whitelist *storage.ProcessWhitelist) error {
	if whitelist.GetId() == "" {
		return fmt.Errorf("whitelist doesn't have an id; key was %+v", whitelist.GetKey())
	}
	return b.Update(func(tx *bolt.Tx) error {
		return addWhitelist([]byte(whitelist.GetId()), whitelist, tx)
	})
}

func updateWhitelist(id []byte, whitelist *storage.ProcessWhitelist, tx *bolt.Tx) error {
	bucket := tx.Bucket(processWhitelistBucket)
	if exists := bolthelper.ExistsBytes(bucket, id); !exists {
		return fmt.Errorf("updating non-existent whitelist: %s", string(id))
	}
	whitelist.LastUpdate = types.TimestampNow()
	return storeWhitelist(id, whitelist, bucket)
}

func (b *storeImpl) UpdateWhitelist(whitelist *storage.ProcessWhitelist) error {
	return b.Update(func(tx *bolt.Tx) error {
		return updateWhitelist([]byte(whitelist.GetId()), whitelist, tx)
	})
}

func (b *storeImpl) deleteWhitelist(tx *bolt.Tx, id []byte) (bool, error) {
	bucket := tx.Bucket(processWhitelistBucket)
	if exists := bolthelper.ExistsBytes(bucket, id); !exists {
		return false, nil
	}
	return true, bucket.Delete(id)
}

func (b *storeImpl) DeleteWhitelist(id string) (bool, error) {
	var exists bool
	err := b.Update(func(tx *bolt.Tx) error {
		var err error
		exists, err = b.deleteWhitelist(tx, []byte(id))
		return err
	})

	return exists, err
}
