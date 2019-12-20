package crud

import (
	"encoding/binary"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/conv"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/sequence"
)

var (
	transactionPrefix = []byte("transactions")
)

// NewTxnCounter returns a TxnCounter implementation with its initial value set from the DB.
func NewTxnCounter(duckBox *dackbox.DackBox, objectType []byte) (*TxnCounter, error) {
	counter := &TxnCounter{
		duckBox: duckBox,
		key:     append(transactionPrefix, objectType...),
		currVal: sequence.NewSequence(),
	}
	return counter, counter.getInitialValue()
}

// TxnCounter provides functionality for tracking a persisted transaction count.
type TxnCounter struct {
	duckBox *dackbox.DackBox
	key     []byte
	currVal sequence.Sequence
}

// IncTxnCount increases the number of transactions for the specific object type
func (b *TxnCounter) IncTxnCount() error {
	return b.duckBox.AtomicKVUpdate(func() (key, value []byte) {
		return b.key, conv.Itob(b.currVal.Add())
	})
}

// GetTxnCount retrieves the number of transactions for the specific object type
func (b *TxnCounter) GetTxnCount() uint64 {
	return b.currVal.Load()
}

func (b *TxnCounter) getInitialValue() error {
	branch := b.duckBox.NewReadOnlyTransaction()
	defer branch.Discard()

	var value uint64
	item, err := branch.BadgerTxn().Get(b.key)
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
	b.currVal.Set(value)
	return nil
}
