package graph

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
)

// NewPersistedGraph returns a new instance of a PersistedGraph.
func NewPersistedGraph(prefix []byte, txn *badger.Txn, base RWGraph) *PersistedGraph {
	return &PersistedGraph{
		RWGraph: base,
		prefix:  prefix,
		txn:     txn,
	}
}

// PersistedGraph is a graph that writes all modifications to a badger transaction.
type PersistedGraph struct {
	RWGraph

	prefix []byte
	txn    *badger.Txn
}

// SetRefs overrides the SetGraph of the parent Branch and writes any changes to the underlying badger Txn.
func (prv *PersistedGraph) SetRefs(from []byte, to [][]byte) error {
	if err := prv.RWGraph.SetRefs(from, to); err != nil {
		return err
	}
	return prv.storeKey(from)
}

// AddRefs overrides the AddRefs of the parent Branch and writes any changes to the underlying badger Txn.
func (prv *PersistedGraph) AddRefs(from []byte, to ...[]byte) error {
	if err := prv.RWGraph.AddRefs(from, to...); err != nil {
		return err
	}
	return prv.storeKey(from)
}

// DeleteRefs overrides the DeleteRefs of the parent Branch and writes any changes to the underlying badger Txn.
func (prv *PersistedGraph) DeleteRefs(from []byte) error {
	if err := prv.RWGraph.DeleteRefs(from); err != nil {
		return err
	}
	return prv.deleteKey(from)
}

func (prv *PersistedGraph) storeKey(from []byte) error {
	return prv.txn.Set(prv.prefixKey(from), sortedkeys.SortedKeys(prv.RWGraph.GetRefsFrom(from)).Marshal())
}

func (prv *PersistedGraph) deleteKey(from []byte) error {
	return prv.txn.Delete(prv.prefixKey(from))
}

func (prv *PersistedGraph) prefixKey(key []byte) []byte {
	return badgerhelper.GetBucketKey(prv.prefix, key)
}
