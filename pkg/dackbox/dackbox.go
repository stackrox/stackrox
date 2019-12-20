package dackbox

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stackrox/rox/pkg/sync"
)

// NewDackBox returns a new DackBox object using the given DB and prefix for storing data and ids.
func NewDackBox(db *badger.DB, graphPrefix []byte) (*DackBox, error) {
	initial, err := loadIntoMem(db, graphPrefix)
	if err != nil {
		return nil, err
	}
	ret := &DackBox{
		history: graph.NewHistory(initial),
		db:      db,
		prefix:  graphPrefix,
	}
	return ret, nil
}

// DackBox is the StackRox DB layer. It provides transactions consisting of both a KV layer, and an ID->[]ID map layer.
type DackBox struct {
	lock sync.RWMutex

	prefix  []byte
	db      *badger.DB
	history graph.History
}

// NewTransaction returns a new Transaction object for read and write operations on both key/value pairs, and ids.
func (rc *DackBox) NewTransaction() *Transaction {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	ts := rc.history.Hold()
	txn := rc.db.NewTransactionAt(ts, true)
	modified := graph.NewModifiedGraph(graph.NewRemoteGraph(graph.NewGraph(), rc.readerAt(ts)))
	return &Transaction{
		ts:           ts,
		txn:          txn,
		graph:        graph.NewPersistedGraph(rc.prefix, txn, modified),
		modification: modified,
		discard:      rc.discard,
		commit:       rc.commit,
	}
}

// NewReadOnlyTransaction returns a Transaction object for read only operations.
func (rc *DackBox) NewReadOnlyTransaction() *Transaction {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	ts := rc.history.Hold()
	txn := rc.db.NewTransactionAt(ts, false)
	modified := graph.NewModifiedGraph(graph.NewRemoteGraph(graph.NewGraph(), rc.readerAt(ts)))
	return &Transaction{
		ts:           ts,
		txn:          txn,
		graph:        graph.NewPersistedGraph(rc.prefix, txn, modified),
		modification: modified,
		discard:      rc.discard,
		commit:       rc.commit,
	}
}

// NewGraphView returns a read only view of the ID->[]ID graph.
func (rc *DackBox) NewGraphView() DiscardableRGraph {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	ts := rc.history.Hold()
	return &discardableGraphImpl{
		ts:          ts,
		RemoteGraph: graph.NewRemoteGraph(graph.NewGraph(), rc.readerAt(ts)),
		discard:     rc.discard,
	}
}

// AtomicKVUpdate updates a key:value pair in badger atomically. It calls the input function inside the global lock,
// so only use where the input function is very fast.
func (rc *DackBox) AtomicKVUpdate(provide func() (key, value []byte)) error {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	ts := rc.history.StepForward()
	txn := rc.db.NewTransactionAt(ts-1, true)
	defer txn.Discard()
	if err := txn.Set(provide()); err != nil {
		return err
	}
	return txn.CommitAt(ts, nil)
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

	if txn != nil {
		txn.Discard()
	}
	rc.history.Release(openedAt)
}

func (rc *DackBox) commit(openedAt uint64, txn *badger.Txn, modification graph.Modification) error {
	rc.lock.Lock()
	defer rc.lock.Unlock()

	ts := rc.history.StepForward()
	if txn != nil {
		if err := txn.CommitAt(ts, nil); err != nil {
			return err
		}
	}
	rc.history.Apply(modification)
	rc.history.Release(openedAt)
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

func loadIntoMem(db *badger.DB, graphPrefix []byte) (*graph.Graph, error) {
	initial := graph.NewGraph()
	err := badgerhelper.BucketForEach(db.NewTransactionAt(0, false), graphPrefix, onLoadForEachOptions, func(k, v []byte) error {
		sk, err := sortedkeys.Unmarshal(v)
		if err != nil {
			return err
		}
		return initial.SetRefs(k, sk)
	})
	if err != nil {
		return nil, err
	}
	return initial, nil
}
