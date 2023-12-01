package dackboxhelper

import (
	"log"

	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	// GraphBucket specifies the prefix for the id map DackBox tracks and stores in the DB.
	GraphBucket = []byte("dackbox_graph")
	// DirtyBucket specifies the prefix for the set of dirty keys (need re-indexing) to add to dackbox.
	DirtyBucket = []byte("dackbox_dirty")
	// ReindexIfMissingBucket is a bucket for all of the child buckets that do not need reindexing.
	ReindexIfMissingBucket = []byte("dackbox_reindex")

	toIndex queue.WaitableQueue

	globalKeyLock concurrency.KeyFence

	dackBox *dackbox.DackBox

	initialized sync.Once
)

// GetMigrationDackBox returns the migration dackbox.DackBox instance.
func GetMigrationDackBox() *dackbox.DackBox {
	initialize()
	return dackBox
}

// GetMigrationIndexQueue returns the queue of items waiting to be indexed.
func GetMigrationIndexQueue() queue.WaitableQueue {
	initialize()
	return toIndex
}

// GetMigrationKeyFence returns the migration key fence.
func GetMigrationKeyFence() concurrency.KeyFence {
	initialize()
	return globalKeyLock
}

func initialize() {
	initialized.Do(func() {
		toIndex = queue.NewWaitableQueue()
		globalKeyLock = concurrency.NewKeyFence()

		var err error
		dackBox, err = dackbox.NewRocksDBDackBox(rockshelper.GetRocksDB(), toIndex, GraphBucket, DirtyBucket, ReindexIfMissingBucket)
		if err != nil {
			log.Panicf("could not load stored indices: %v", err)
		}
	})
}
