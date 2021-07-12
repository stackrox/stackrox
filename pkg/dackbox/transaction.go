package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/transactions"
	"github.com/stackrox/rox/pkg/dbhelper"
)

var emptyByte = []byte{0}

// RemoteDiscard is a function that discards any changes made in the transaction
type RemoteDiscard func(openedAt uint64, txn transactions.DBTransaction)

// RemoteCommit is a function that can be used to commit a change to DackBox.
type RemoteCommit func(openedAt uint64, txn transactions.DBTransaction, modification graph.Modification, dirtyKeys map[string]proto.Message) error

// Transaction is a linked graph and database transaction.
type Transaction struct {
	ts uint64

	transactions.DBTransaction

	graph        *graph.RemoteGraph
	modification graph.Modification

	dirtyPrefix []byte
	dirtyMap    map[string]proto.Message

	closed  bool
	discard RemoteDiscard
	commit  RemoteCommit
}

// Graph returns the Graph object (the ID->[]ID mapping layer) in the current transaction.
func (dbt *Transaction) Graph() *graph.RemoteGraph {
	return dbt.graph
}

// MarkDirty adds the input key to the dirty set, and adds he key and value to the queue for indexing.
func (dbt *Transaction) MarkDirty(key []byte, msg proto.Message) {
	dbt.Set(dbhelper.GetBucketKey(dbt.dirtyPrefix, key), emptyByte)
	dbt.dirtyMap[string(key)] = msg
}

// BaseTS returns the time-step the transaction was created at.
func (dbt *Transaction) BaseTS() uint64 {
	return dbt.ts
}

// Discard  dumps all of the transaction's changes.
func (dbt *Transaction) Discard() {
	if dbt.closed {
		return
	}
	dbt.closed = true
	dbt.discard(dbt.ts, dbt.DBTransaction)
}

// Commit the transaction's changes to the remote graph.
func (dbt *Transaction) Commit() error {
	dbt.closed = true
	return dbt.commit(dbt.ts, dbt.DBTransaction, dbt.modification, dbt.dirtyMap)
}
