package store

import (
	"fmt"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
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

func storeWhitelist(whitelist *storage.ProcessWhitelist, bucket *bolt.Bucket) error {
	bytes, err := proto.Marshal(whitelist)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(whitelist.GetId()), bytes)
}

func addWhitelist(whitelist *storage.ProcessWhitelist, tx *bolt.Tx) error {
	bucket := tx.Bucket(processWhitelistBucket)
	exists := bolthelper.Exists(bucket, whitelist.GetId())
	if exists {
		return errors.New(fmt.Sprintf("whitelist %s already exists", whitelist.GetId()))
	}
	return storeWhitelist(whitelist, bucket)
}

func (b *storeImpl) AddWhitelist(whitelist *storage.ProcessWhitelist) error {
	if whitelist.GetId() == "" {
		return fmt.Errorf("tried to store a whitelist with no ID.  Deployment ID: %q, Container Name: %q", whitelist.DeploymentId, whitelist.ContainerName)
	}
	return b.Update(func(tx *bolt.Tx) error {
		return addWhitelist(whitelist, tx)
	})
}

func updateWhitelist(whitelist *storage.ProcessWhitelist, tx *bolt.Tx) error {
	bucket := tx.Bucket(processWhitelistBucket)
	exists := bolthelper.Exists(bucket, whitelist.GetId())
	if !exists {
		return errors.New(fmt.Sprintf("updating non-existant whitelist %s", whitelist.GetId()))
	}
	return storeWhitelist(whitelist, bucket)
}

func (b *storeImpl) UpdateWhitelist(whitelist *storage.ProcessWhitelist) error {
	return b.Update(func(tx *bolt.Tx) error {
		return updateWhitelist(whitelist, tx)
	})
}

func (b *storeImpl) deleteWhitelist(tx *bolt.Tx, id string) (bool, error) {
	bucket := tx.Bucket(processWhitelistBucket)
	if exists := bolthelper.Exists(bucket, id); !exists {
		return false, nil
	}
	key := []byte(id)
	return true, bucket.Delete(key)
}

func (b *storeImpl) DeleteWhitelist(id string) (bool, error) {
	var exists bool
	err := b.Update(func(tx *bolt.Tx) error {
		var err error
		exists, err = b.deleteWhitelist(tx, id)
		return err
	})

	return exists, err
}
