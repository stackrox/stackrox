package connection

import (
	"context"
	"hash/fnv"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

type workerQueue struct {
	poolSize  int
	totalSize int

	queues    []*dedupingQueue
	waitGroup sync.WaitGroup
	sync.WaitGroup
}

func newWorkerQueue(poolSize int, typ string) *workerQueue {
	totalSize := poolSize + 1
	queues := make([]*dedupingQueue, totalSize)
	for i := 0; i < totalSize; i++ {
		queues[i] = newDedupingQueue(typ)
	}

	return &workerQueue{
		poolSize:  poolSize,
		totalSize: totalSize,
		queues:    queues,
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
	if msg.GetEvent().GetCheckpoint() != nil {
		for _, queue := range w.queues {
			queue.push(msg)
		}
		return
	}

	// The zeroth index is reserved for objects that do not match the switch statement below
	// w.indexFromKey returns (hashed value % poolSize) + 1 so it cannot return a 0 index
	var idx int
	if msg.HashKey != "" {
		idx = w.indexFromKey(msg.HashKey)
	}

	w.queues[idx].push(msg)
}

func (w *workerQueue) runWorker(ctx context.Context, idx int, stopSig *concurrency.ErrorSignal, handler func(context.Context, *central.MsgFromSensor) error) {
	queue := w.queues[idx]
	for msg := queue.pullBlocking(stopSig); msg != nil; msg = queue.pullBlocking(stopSig) {
		if checkpoint := msg.GetEvent().GetCheckpoint(); checkpoint != nil {
			MarkCheckpoint(checkpoint.GetId())
			continue
		}
		if err := handler(ctx, msg); err != nil {
			log.Errorf("Error handling sensor message: %v", err)
		}
	}
	w.waitGroup.Add(-1)
}

func (w *workerQueue) run(ctx context.Context, stopSig *concurrency.ErrorSignal, handler func(context.Context, *central.MsgFromSensor) error) {
	w.waitGroup.Add(w.totalSize)
	for i := 0; i < w.totalSize; i++ {
		go w.runWorker(ctx, i, stopSig, handler)
	}

	w.waitGroup.Wait()
}
