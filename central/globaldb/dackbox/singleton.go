package dackbox

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	// GraphBucket specifies the prefix for the id map DackBox tracks and stores in the DB.
	GraphBucket = []byte("dackbox_graph")
	// DirtyBucket specifies the prefix for the set of dirty keys (need re-indexing) to add to dackbox.
	DirtyBucket = []byte("dackbox_dirty")
	// ReindexIfMissingBucket is a bucket for all of the child buckets that do not need reindexing.
	ReindexIfMissingBucket = []byte("dackbox_reindex")
	needsReindexValue      = []byte{0}

	toIndex  queue.WaitableQueue
	registry indexer.WrapperRegistry
	lazy     indexer.Lazy

	globalKeyLock concurrency.KeyFence

	duckBox *dackbox.DackBox

	dackBoxInit sync.Once
)

// GetGlobalDackBox returns the global dackbox.DackBox instance.
func GetGlobalDackBox() *dackbox.DackBox {
	postgres.LogCallerOnPostgres("GetGlobalDackBox")
	initializeDackBox()
	return duckBox
}

// GetIndexQueue returns the queue of items waiting to be indexed.
func GetIndexQueue() queue.WaitableQueue {
	initializeDackBox()
	return toIndex
}

// GetKeyFence returns the global key fence.
func GetKeyFence() concurrency.KeyFence {
	initializeDackBox()
	return globalKeyLock
}

func initializeDackBox() {
	dackBoxInit.Do(func() {
		globaldb.RegisterBucket(GraphBucket, "Graph Keys")
		globaldb.RegisterBucket(DirtyBucket, "Dirty Keys")
		globaldb.RegisterBucket(ReindexIfMissingBucket, "Bucket for reindexed state")

		toIndex = queue.NewWaitableQueue()
		registry = indexer.NewWrapperRegistry()
		globalKeyLock = concurrency.NewKeyFence()

		var err error
		duckBox, err = dackbox.NewRocksDBDackBox(globaldb.GetRocksDB(), toIndex, GraphBucket, DirtyBucket, ReindexIfMissingBucket)
		if err != nil {
			log.Panicf("could not load stored indices: %v", err)
		}

		lazy = indexer.NewLazy(toIndex, registry, globalindex.GetGlobalIndex(), duckBox.AckIndexed)
		lazy.Start()

		if err := Init(duckBox, toIndex, registry, ReindexIfMissingBucket, DirtyBucket, needsReindexValue); err != nil {
			log.Panicf("Could not initialize dackbox: %v", err)
		}
	})
}
