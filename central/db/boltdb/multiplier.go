package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const multiplierBucket = "multipliers"

func (b *BoltDB) getMultiplier(id string, bucket *bolt.Bucket) (multiplier *v1.Multiplier, exists bool, err error) {
	multiplier = new(v1.Multiplier)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, multiplier)
	return
}

// GetMultiplier returns multiplier with given id.
func (b *BoltDB) GetMultiplier(id string) (multiplier *v1.Multiplier, exists bool, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(multiplierBucket))
		multiplier, exists, err = b.getMultiplier(id, bucket)
		return err
	})
	return
}

// GetMultipliers retrieves multipliers from bolt
func (b *BoltDB) GetMultipliers() ([]*v1.Multiplier, error) {
	var multipliers []*v1.Multiplier
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(multiplierBucket))
		return b.ForEach(func(k, v []byte) error {
			var multiplier v1.Multiplier
			if err := proto.Unmarshal(v, &multiplier); err != nil {
				return err
			}
			multipliers = append(multipliers, &multiplier)
			return nil
		})
	})
	return multipliers, err
}

// AddMultiplier adds a multiplier into bolt
func (b *BoltDB) AddMultiplier(multiplier *v1.Multiplier) (string, error) {
	multiplier.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(multiplierBucket))
		_, exists, err := b.getMultiplier(multiplier.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Multiplier %s (%s) cannot be added because it already exists", multiplier.GetId(), multiplier.GetName())
		}
		if err := checkUniqueKeyExistsAndInsert(tx, multiplierBucket, multiplier.GetId(), multiplier.GetName()); err != nil {
			return fmt.Errorf("Could not add multiplier due to name validation: %s", err)
		}
		bytes, err := proto.Marshal(multiplier)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(multiplier.GetId()), bytes)
	})
	return multiplier.Id, err
}

// UpdateMultiplier upserts a multiplier into bolt
func (b *BoltDB) UpdateMultiplier(multiplier *v1.Multiplier) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(multiplierBucket))
		// If the update is changing the name, check if the name has already been taken
		if getCurrentUniqueKey(tx, multiplierBucket, multiplier.GetId()) != multiplier.GetName() {
			if err := checkUniqueKeyExistsAndInsert(tx, multiplierBucket, multiplier.GetId(), multiplier.GetName()); err != nil {
				return fmt.Errorf("Could not update multiplier due to name validation: %s", err)
			}
		}
		bytes, err := proto.Marshal(multiplier)
		if err != nil {
			return err
		}
		return b.Put([]byte(multiplier.GetId()), bytes)
	})
}

// RemoveMultiplier removes a multiplier from bolt
func (b *BoltDB) RemoveMultiplier(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(multiplierBucket))
		key := []byte(id)
		if exists := b.Get(key) != nil; !exists {
			return db.ErrNotFound{Type: "Multiplier", ID: string(key)}
		}
		if err := removeUniqueKey(tx, multiplierBucket, id); err != nil {
			return err
		}
		return b.Delete(key)
	})
}
