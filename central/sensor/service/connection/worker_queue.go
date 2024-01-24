package connection

import (
	"context"
	"hash/fnv"
	"time"

	"github.com/pkg/errors"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	maxHandlerAttempts   = 5
	handlerRetryInterval = 5 * time.Minute
)

type workerQueue struct {
	poolSize  int
	totalSize int

	injector  common.MessageInjector
	queues    []*dedupingQueue
	waitGroup sync.WaitGroup
	sync.WaitGroup
}

func newWorkerQueue(poolSize int, typ string, injector common.MessageInjector) *workerQueue {
	totalSize := poolSize + 1
	queues := make([]*dedupingQueue, totalSize)
	for i := 0; i < totalSize; i++ {
		queues[i] = newDedupingQueue(typ)
	}

	return &workerQueue{
		poolSize:  poolSize,
		totalSize: totalSize,
		queues:    queues,
		injector:  injector,
	}
}

func (w *workerQueue) indexFromKey(key string) int {
	h := fnv.New32()
	// Write never returns an error.
	_, _ = h.Write([]byte(key))
	// Increment by one because the zero index is reserved for non sharded objects
	return (int(h.Sum32()) % w.poolSize) + 1
}

// push attempts to add an item to the queue, and returns an error if it is unable.
func (w *workerQueue) push(msg *central.MsgFromSensor) {
	// The zeroth index is reserved for objects that do not match the switch statement below
	// w.indexFromKey returns (hashed value % poolSize) + 1 so it cannot return a 0 index
	var idx int
	if msg.HashKey != "" {
		idx = w.indexFromKey(msg.HashKey)
	}

	w.queues[idx].push(msg)
}

func (w *workerQueue) runWorker(ctx context.Context, idx int, stopSig *concurrency.ErrorSignal, deduper hashManager.Deduper, handler func(context.Context, *central.MsgFromSensor) error) {
	queue := w.queues[idx]
	for msg := queue.pullBlocking(stopSig); msg != nil; msg = queue.pullBlocking(stopSig) {
		err := handler(ctx, msg)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				if pgutils.IsTransientError(err) {
					msg.ProcessingAttempt++

					if msg.ProcessingAttempt == maxHandlerAttempts {
						log.Errorf("Error handling sensor message %T permanently: %v", msg.GetEvent().GetResource(), err)
						continue
					}
					reprocessingDuration := time.Duration(msg.ProcessingAttempt) * handlerRetryInterval
					log.Warnf("Reprocessing sensor message %T in %d minutes: %v", msg.GetEvent().GetResource(), int(reprocessingDuration.Minutes()), err)
					concurrency.AfterFunc(reprocessingDuration, func() {
						w.injector.InjectMessageIntoQueue(msg)
					}, stopSig)
					continue
				}
				log.Errorf("Unretryable error found while handling sensor message %T permanently: %v", msg.GetEvent().GetResource(), err)
			}
			deduper.RemoveMessage(msg)
		}
	}
	w.waitGroup.Add(-1)
}

func (w *workerQueue) run(ctx context.Context, stopSig *concurrency.ErrorSignal, deduper hashManager.Deduper, handler func(context.Context, *central.MsgFromSensor) error) {
	w.waitGroup.Add(w.totalSize)
	for i := 0; i < w.totalSize; i++ {
		go w.runWorker(ctx, i, stopSig, deduper, handler)
	}

	w.waitGroup.Wait()
}
