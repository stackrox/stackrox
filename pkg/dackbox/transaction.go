package dackbox

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/dackbox/graph"
)

// RemoteDiscard is a function that discards any changes made in the transaction
type RemoteDiscard func(openedAt uint64, txn *badger.Txn)

// RemoteCommit is a function that can be used to commit a change to DackBox.
type RemoteCommit func(openedAt uint64, txn *badger.Txn, modification graph.Modification) error

// Transaction is a linked graph and badger transaction.
type Transaction struct {
	ts uint64

	txn          *badger.Txn
	graph        *graph.PersistedGraph
	modification graph.Modification

	discard RemoteDiscard
	commit  RemoteCommit
}

// BadgerTxn returns the badger Txn object for making read and write requests to the KV layer.
func (dbt *Transaction) BadgerTxn() *badger.Txn {
	return dbt.txn
}

// Graph returns the Graph object (the ID->[]ID mapping layer) in the current transaction.
func (dbt *Transaction) Graph() *graph.PersistedGraph {
	return dbt.graph
}

// BaseTS returns the time-step the transaction was created at.
func (dbt *Transaction) BaseTS() uint64 {
	return dbt.ts
}

// Discard  dumps all of the transaction's changes.
func (dbt *Transaction) Discard() {
	dbt.discard(dbt.ts, dbt.txn)
}

// Commit the transaction's changes to the remote graph.
func (dbt *Transaction) Commit() error {
	return dbt.commit(dbt.ts, dbt.txn, dbt.modification)
}
