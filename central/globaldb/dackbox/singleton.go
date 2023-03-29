package dackbox

import (
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	rawDackbox "github.com/stackrox/rox/pkg/dackbox/raw"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	needsReindexValue = []byte{0}
	dackBoxInit       sync.Once
)

// GetGlobalDackBox returns the global dackbox.DackBox instance.
func GetGlobalDackBox() *dackbox.DackBox {
	postgres.DeprecatedCall("GetGlobalDackBox")
	initializeDackBox()
	return rawDackbox.GetGlobalDackBox()
}

// GetIndexQueue returns the queue of items waiting to be indexed.
func GetIndexQueue() queue.WaitableQueue {
	postgres.DeprecatedCall("GetIndexQueue")
	initializeDackBox()
	return rawDackbox.GetIndexQueue()
}

// GetKeyFence returns the global key fence.
func GetKeyFence() concurrency.KeyFence {
	postgres.DeprecatedCall("GetKeyFence")
	initializeDackBox()
	return rawDackbox.GetKeyFence()
}

func initializeDackBox() {
	dackBoxInit.Do(func() {
		rawDackbox.StartIndexer(globalindex.GetGlobalIndex())
		if err := Init(rawDackbox.GetGlobalDackBox(), rawDackbox.GetIndexQueue(), rawDackbox.DirtyBucket); err != nil {
			log.Panicf("Could not initialize dackbox: %v", err)
		}
	})
}
