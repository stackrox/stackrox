package blevehelper

import (
	"encoding/binary"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/pkg/conv"
	"github.com/stackrox/rox/pkg/sync"
)

// GlobalIndexName helps differentiate between temporary indexes and the global index
const GlobalIndexName = "global"

// NewBleveWrapper returns a bleve wrapper that implements transactional support
func NewBleveWrapper(index bleve.Index, objectType string) (*BleveWrapper, error) {
	wrapper := &BleveWrapper{
		Index:      index,
		isGlobalDB: index.Name() == GlobalIndexName,

		objectType:      objectType,
		objectTypeBytes: []byte(objectType),
	}
	currVal, err := wrapper.readInitialValue()
	if err != nil {
		return nil, err
	}
	wrapper.currVal = currVal
	return wrapper, nil
}

// BleveWrapper implements
type BleveWrapper struct {
	bleve.Index
	isGlobalDB bool

	objectType      string
	objectTypeBytes []byte

	lock    sync.RWMutex
	currVal uint64
}

func (w *BleveWrapper) readInitialValue() (uint64, error) {
	value, err := w.Index.GetInternal(w.objectTypeBytes)
	if err != nil {
		return 0, err
	}
	if value == nil {
		return 0, nil
	}
	return binary.BigEndian.Uint64(value), nil
}

// IncTxnCount increments the transaction count for the specific object type
func (w *BleveWrapper) IncTxnCount() error {
	if !w.isGlobalDB {
		return nil
	}

	w.lock.Lock()
	defer w.lock.Unlock()

	w.currVal++

	return w.setTxnCountNoLock(w.currVal)
}

// GetTxnCount returns the total number of transactions for that object type
func (w *BleveWrapper) GetTxnCount() uint64 {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.currVal
}

func (w *BleveWrapper) setTxnCountNoLock(txNum uint64) error {
	return w.SetInternal(w.objectTypeBytes, conv.Itob(txNum))
}

// SetTxnCount clears the transaction count after reconciliation
func (w *BleveWrapper) SetTxnCount(txNum uint64) error {
	if !w.isGlobalDB {
		return nil
	}

	w.lock.Lock()
	defer w.lock.Unlock()
	return w.setTxnCountNoLock(txNum)
}
