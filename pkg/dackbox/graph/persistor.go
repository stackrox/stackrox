package graph

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// NewPersistor returns a new instance of a Persistor, which can be used to apply modifications to the persisted graph.
func NewPersistor(prefix []byte, txn *badger.Txn) *Persistor {
	return &Persistor{
		prefix: prefix,
		txn:    txn,
	}
}

// Persistor is an applyableGraph implementation that persists what is applied to it.
type Persistor struct {
	prefix []byte
	txn    *badger.Txn
	errors errorhelpers.ErrorList
}

// ToError returns an error if any errors were encountered when persisting the modification it was applied to.
func (prv *Persistor) ToError() error {
	return prv.errors.ToError()
}

// Implement applyableGraph.
func (prv *Persistor) setFrom(from []byte, to [][]byte) {
	prv.errors.AddError(prv.txn.Set(prv.prefixKey(from), sortedkeys.SortedKeys(to).Marshal()))
}

func (prv *Persistor) deleteFrom(from []byte) {
	prv.errors.AddError(prv.txn.Delete(prv.prefixKey(from)))
}

func (prv *Persistor) setTo(to []byte, from [][]byte) {
	// do nothing, we only store the forward map.
}

func (prv *Persistor) deleteTo(to []byte) {
	// do nothing, we only store the forward map.
}

func (prv *Persistor) prefixKey(key []byte) []byte {
	return badgerhelper.GetBucketKey(prv.prefix, key)
}
