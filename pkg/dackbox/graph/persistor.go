package graph

import (
	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stackrox/rox/pkg/dackbox/transactions"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// NewPersistor returns a new instance of a Persistor, which can be used to apply modifications to the persisted graph.
func NewPersistor(prefix []byte, txn transactions.DBTransaction) *Persistor {
	return &Persistor{
		prefix: prefix,
		txn:    txn,
	}
}

// Persistor is an applyableGraph implementation that persists what is applied to it.
type Persistor struct {
	prefix []byte
	txn    transactions.DBTransaction
	errors errorhelpers.ErrorList
}

// ToError returns an error if any errors were encountered when persisting the modification it was applied to.
func (prv *Persistor) ToError() error {
	return prv.errors.ToError()
}

// Implement applyableGraph.
func (prv *Persistor) setFrom(from []byte, to [][]byte) {
	prv.txn.Set(prv.prefixKey(from), sortedkeys.SortedKeys(to).Marshal())
}

func (prv *Persistor) deleteFrom(from []byte) {
	prv.txn.Delete(prv.prefixKey(from))
}

func (prv *Persistor) setTo(_ []byte, _ [][]byte) {
	// do nothing, we only store the forward map.
}

func (prv *Persistor) deleteTo(_ []byte) {
	// do nothing, we only store the forward map.
}

func (prv *Persistor) prefixKey(key []byte) []byte {
	return dbhelper.GetBucketKey(prv.prefix, key)
}
