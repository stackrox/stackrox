package dackbox

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stackrox/rox/pkg/dackbox/transactions"
	badgerTxns "github.com/stackrox/rox/pkg/dackbox/transactions/badger"
	"github.com/stackrox/rox/pkg/dackbox/transactions/rocksdb"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/tecbot/gorocksdb"
)

func newDackBox(dbFactory transactions.DBTransactionFactory, toIndex queue.AcceptsKeyValue, graphPrefix, dirtyPrefix, validPrefix []byte) (*DackBox, error) {
	initial, err := loadGraphIntoMem(dbFactory, graphPrefix)
	if err != nil {
		return nil, err
	}
	ret := &DackBox{
		history:     graph.NewHistory(initial),
		db:          dbFactory,
		toIndex:     toIndex,
		graphPrefix: graphPrefix,
		dirtyPrefix: dirtyPrefix,
		validPrefix: validPrefix,
	}
	return ret, nil
}

// NewDackBox returns a new DackBox object using the given DB and prefix for storing data and ids.
func NewDackBox(db *badger.DB, toIndex queue.AcceptsKeyValue, graphPrefix, dirtyPrefix, validPrefix []byte) (*DackBox, error) {
	return newDackBox(badgerTxns.NewBadgerWrapper(db), toIndex, graphPrefix, dirtyPrefix, validPrefix)
}

// NewRocksDBDackBox creates an instance of dackbox based on RocksDB
func NewRocksDBDackBox(db *gorocksdb.DB, toIndex queue.AcceptsKeyValue, graphPrefix, dirtyPrefix, validPrefix []byte) (*DackBox, error) {
	return newDackBox(rocksdb.NewRocksDBWrapper(db), toIndex, graphPrefix, dirtyPrefix, validPrefix)
}

// DackBox is the StackRox DB layer. It provides transactions consisting of both a KV layer, and an ID->[]ID map layer.
type DackBox struct {
	lock sync.RWMutex

	history graph.History
	db      transactions.DBTransactionFactory
	toIndex queue.AcceptsKeyValue

	graphPrefix []byte
	dirtyPrefix []byte
	validPrefix []byte
}

// NewTransaction returns a new Transaction object for read and write operations on both key/value pairs, and ids.
func (rc *DackBox) NewTransaction() *Transaction {
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	// Hold the current state of the graph for the transaction.
	ts := rc.history.Hold()
	// Create a read-write DB transaction.
	txn := rc.db.NewTransaction(true)

	// Create a graph modification. This will record changes made to a new, empty, underlying graph.
	modification := graph.NewModifiedGraph(graph.NewGraph())
	// Wrap the modification with a remote graph, this will pull the data from the graph history when values are read or
	// written, and push any changes made into the modification where they will be recorded.
	remote := graph.NewRemoteGraph(modification, rc.readerAt(ts))

	// Return the constructed transaction.
	return &Transaction{
		ts:            ts,
		DBTransaction: txn,
		graph:         remote,
		modification:  modification,
		dirtyPrefix:   rc.dirtyPrefix,
		dirtyMap:      make(map[string]proto.Message),
		discard:       rc.discard,
		commit:        rc.commit,
	}
}

// NewReadOnlyTransaction returns a Transaction object for read only operations.
func (rc *DackBox) NewReadOnlyTransaction() *Transaction {
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	// Hold the current state of the graph for the transaction.
	ts := rc.history.Hold()
	// Create a read-only DB transaction.
	txn := rc.db.NewTransaction(false)

	// Wrap an empty graph with a remote graph. It will serve as a cache to store values read from the history.
	remote := graph.NewRemoteGraph(graph.NewGraph(), rc.readerAt(ts))

	// Return the constructed transaction.
	return &Transaction{
		ts:            ts,
		DBTransaction: txn,
		graph:         remote,
		modification:  nil,
		discard:       rc.discard,
		commit:        rc.commit,
	}
}

// NewGraphView returns a read only view of the ID->[]ID graph.
func (rc *DackBox) NewGraphView() graph.DiscardableRGraph {
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	ts := rc.history.Hold()
	return graph.NewDiscardableGraph(
		graph.NewRemoteGraph(graph.NewGraph(), rc.readerAt(ts)),
		func() { rc.discard(ts, nil) },
	)
}

// AckIndexed is an exposed way to remove keys from the dirty bucket.
func (rc *DackBox) AckIndexed(keys ...[]byte) error {
	if len(keys) == 0 {
		return nil
	}

	txn := rc.db.NewTransaction(true)
	defer txn.Discard()
	for _, key := range keys {
		if err := txn.Delete(dbhelper.GetBucketKey(rc.dirtyPrefix, key)); err != nil {
			return err
		}
	}
	return txn.Commit()
}

func (rc *DackBox) readerAt(at uint64) graph.RemoteReadable {
	return func(reader func(graph graph.RGraph)) {
		rc.lock.RLock()
		defer rc.lock.RUnlock()

		reader(rc.history.View(at))
	}
}

func (rc *DackBox) discard(openedAt uint64, txn transactions.DBTransaction) {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	// Release the held history no matter what.
	rc.history.Release(openedAt)

	// Discard the disk changes.
	if txn != nil {
		txn.Discard()
	}
}

func (rc *DackBox) commit(openedAt uint64, txn transactions.DBTransaction, modification graph.Modification, dirtyMap map[string]proto.Message) error {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	// Release the held history no matter what.
	rc.history.Release(openedAt)

	// return early if there is no modification associated with the transaction.
	if modification == nil {
		return nil
	}

	// Try to commit the disk changes. Do nothing to the in-memory state if that fails.
	if txn != nil {
		defer txn.Discard()

		// Persist graph updates.
		persistor := graph.NewPersistor(rc.graphPrefix, txn)
		modification.Apply(persistor)
		if err := persistor.ToError(); err != nil {
			return err
		}

		// Commit the transaction with graph updates added.
		if err := txn.Commit(); err != nil {
			return err
		}
	}

	// We should only add to the dirty queue and add the graph modification if the transaction was submitted successfully.
	if rc.toIndex != nil {
		for k, v := range dirtyMap {
			rc.toIndex.Push([]byte(k), v)
		}
	}
	rc.history.Apply(modification)
	return nil
}

// Initialization.
//////////////////

func loadGraphIntoMem(dbFactory transactions.DBTransactionFactory, graphPrefix []byte) (*graph.Graph, error) {
	initial := graph.NewGraph()

	txn := dbFactory.NewTransaction(false)
	defer txn.Discard()

	err := txn.BucketForEach(graphPrefix, true, func(k, v []byte) error {
		sk, err := sortedkeys.Unmarshal(v)
		if err != nil {
			return err
		}
		return initial.SetRefs(append([]byte{}, k...), sk)
	})
	if err != nil {
		return nil, err
	}
	return initial, nil
}
