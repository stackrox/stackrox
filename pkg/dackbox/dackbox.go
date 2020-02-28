package dackbox

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/sync"
)

// NewDackBox returns a new DackBox object using the given DB and prefix for storing data and ids.
func NewDackBox(db *badger.DB, toIndex queue.AcceptsKeyValue, graphPrefix, dirtyPrefix, validPrefix []byte) (*DackBox, error) {
	initial, err := loadGraphIntoMem(db, graphPrefix)
	if err != nil {
		return nil, err
	}
	ret := &DackBox{
		history:     graph.NewHistory(initial),
		db:          db,
		toIndex:     toIndex,
		graphPrefix: graphPrefix,
		dirtyPrefix: dirtyPrefix,
		validPrefix: validPrefix,
	}
	return ret, nil
}

// DackBox is the StackRox DB layer. It provides transactions consisting of both a KV layer, and an ID->[]ID map layer.
type DackBox struct {
	lock sync.RWMutex

	history graph.History
	db      *badger.DB
	toIndex queue.AcceptsKeyValue

	graphPrefix []byte
	dirtyPrefix []byte
	validPrefix []byte
}

// NewTransaction returns a new Transaction object for read and write operations on both key/value pairs, and ids.
func (rc *DackBox) NewTransaction() *Transaction {
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	ts := rc.history.Hold()
	txn := rc.db.NewTransaction(true)
	modification := graph.NewModifiedGraph(graph.NewGraph())
	remote := graph.NewRemoteGraph(modification, rc.readerAt(ts))
	return &Transaction{
		ts:           ts,
		txn:          txn,
		graph:        graph.NewPersistedGraph(rc.graphPrefix, txn, remote),
		modification: modification,
		dirtyPrefix:  rc.dirtyPrefix,
		dirtyMap:     make(map[string]proto.Message),
		discard:      rc.discard,
		commit:       rc.commit,
	}
}

// NewReadOnlyTransaction returns a Transaction object for read only operations.
func (rc *DackBox) NewReadOnlyTransaction() *Transaction {
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	ts := rc.history.Hold()
	txn := rc.db.NewTransaction(false)
	modification := graph.NewModifiedGraph(graph.NewGraph())
	remote := graph.NewRemoteGraph(modification, rc.readerAt(ts))
	return &Transaction{
		ts:           ts,
		txn:          txn,
		graph:        graph.NewPersistedGraph(rc.graphPrefix, txn, remote),
		modification: modification,
		discard:      rc.discard,
		commit:       rc.commit,
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
	return rc.db.Update(func(txn *badger.Txn) error {
		for _, key := range keys {
			if err := txn.Delete(badgerhelper.GetBucketKey(rc.dirtyPrefix, key)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (rc *DackBox) readerAt(at uint64) graph.RemoteReadable {
	return func(reader func(graph graph.RGraph)) {
		rc.lock.RLock()
		defer rc.lock.RUnlock()

		reader(rc.history.View(at))
	}
}

func (rc *DackBox) discard(openedAt uint64, txn *badger.Txn) {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	// Release the held history no matter what.
	rc.history.Release(openedAt)

	// Discard the disk changes.
	if txn != nil {
		txn.Discard()
	}
}

func (rc *DackBox) commit(openedAt uint64, txn *badger.Txn, modification graph.Modification, dirtyMap map[string]proto.Message) error {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	// Release the held history no matter what.
	rc.history.Release(openedAt)

	// Try to commit the disk changes. Do nothing to the in-memory state if that fails.
	if txn != nil {
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

var onLoadForEachOptions = badgerhelper.ForEachOptions{
	IteratorOptions: &badger.IteratorOptions{
		PrefetchValues: true,
		PrefetchSize:   4,
	},
	StripKeyPrefix: true,
}

func loadGraphIntoMem(db *badger.DB, graphPrefix []byte) (*graph.Graph, error) {
	initial := graph.NewGraph()
	err := badgerhelper.BucketForEach(db.NewTransaction(false), graphPrefix, onLoadForEachOptions, func(k, v []byte) error {
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
