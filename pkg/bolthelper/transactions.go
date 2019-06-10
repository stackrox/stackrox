package bolthelper

import (
	"encoding/binary"

	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/pkg/conv"
)

// NewBoltWrapper returns a db wrapper that will increment txn counts
func NewBoltWrapper(db *bbolt.DB, objectType []byte) (*BoltWrapper, error) {
	wrapper := &BoltWrapper{
		DB:         db,
		objectType: objectType,
	}

	val, err := wrapper.getInitialValue()
	if err != nil {
		return nil, err
	}
	wrapper.currVal = val
	return wrapper, nil
}

// BoltWrapper overrides the Update function to increment txn counts
type BoltWrapper struct {
	*bbolt.DB
	objectType []byte

	currVal uint64
}

func (b *BoltWrapper) getInitialValue() (uint64, error) {
	var value uint64
	err := b.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(txnBucket)
		byteVal := bucket.Get(b.objectType)
		if byteVal != nil {
			value = binary.BigEndian.Uint64(byteVal)
		}
		return nil
	})
	return value, err
}

// Update overrides the default update and increments the transaction if there is no error
func (b *BoltWrapper) Update(fn func(*bbolt.Tx) error) error {
	return b.DB.Update(func(tx *bbolt.Tx) error {
		if err := fn(tx); err != nil {
			return err
		}
		return b.IncTxnCount(tx)
	})
}

// IncTxnCount increases the number of transactions for the specific object type
func (b *BoltWrapper) IncTxnCount(tx *bbolt.Tx) error {
	b.currVal++

	bucket := tx.Bucket(txnBucket)
	return bucket.Put(b.objectType, conv.Itob(b.currVal))
}

// GetTxnCount retrieves the number of transactions for the specific object type
func (b *BoltWrapper) GetTxnCount(tx *bbolt.Tx) uint64 {
	return b.currVal
}
