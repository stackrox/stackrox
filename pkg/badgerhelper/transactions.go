package badgerhelper

import (
	"encoding/binary"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/conv"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	transactionPrefix = []byte("transactions")
)

// NewTxnHelper returns a db wrapper that will increment txn counts
func NewTxnHelper(db *badger.DB, objectType []byte) (*TxnHelper, error) {
	wrapper := &TxnHelper{
		db:  db,
		key: append(transactionPrefix, objectType...),
	}

	val, err := wrapper.getInitialValue()
	if err != nil {
		return nil, err
	}
	wrapper.currVal = val
	return wrapper, nil
}

// TxnHelper overrides the Update function to increment txn counts
type TxnHelper struct {
	db *badger.DB

	lock sync.Mutex

	key []byte

	currVal uint64
}

func (b *TxnHelper) getInitialValue() (uint64, error) {
	var value uint64
	err := b.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(b.key)
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		if item != nil {
			if err := item.Value(func(v []byte) error {
				value = binary.BigEndian.Uint64(v)
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return value, err
}

// Merge function to add two uint64 numbers
func inc(existing, new []byte) []byte {
	return conv.Itob(binary.BigEndian.Uint64(existing) + binary.BigEndian.Uint64(new))
}

// IncTxnCount increases the number of transactions for the specific object type
func (b *TxnHelper) IncTxnCount() error {
	m := b.db.GetMergeOperator(b.key, inc, 200*time.Millisecond)
	defer m.Stop()
	if err := m.Add(conv.Itob(1)); err != nil {
		return err
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.currVal++
	return nil
}

// GetTxnCount retrieves the number of transactions for the specific object type
func (b *TxnHelper) GetTxnCount() uint64 {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.currVal
}
