package connection

import (
	"context"
	"hash/fnv"
	"time"

	"github.com/pkg/errors"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dedupingqueue"
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
	queues    []*dedupingqueue.DedupingQueue[string]
	waitGroup sync.WaitGroup
	sync.WaitGroup
}

// newWorkerQueue creates a worker queue of poolSize+1 sharded DedupingQueues for the given
// SensorEvent worker type. The shards are named after the type (stripped of its prefix) and
// each shard caps its buffered messages at maxShardSizeForType(workerType) (0 means
// unbounded). The cap is per shard rather than a shared total so that one busy shard cannot
// evict messages queued for other shards; see pkg/env/alert_queue.go for why only the alerts
// queue is capped today.
func newWorkerQueue(poolSize int, workerType string, injector common.MessageInjector) *workerQueue {
	totalSize := poolSize + 1
	name := stripTypePrefix(workerType)
	maxShardSize := maxShardSizeForType(workerType)
	queues := make([]*dedupingqueue.DedupingQueue[string], totalSize)
	for i := range totalSize {
		queues[i] = dedupingqueue.NewDedupingQueue[string](
			dedupingqueue.WithQueueName[string](name),
			dedupingqueue.WithMaxSize[string](maxShardSize),
			dedupingqueue.WithOperationMetricsFunc[string](metrics.IncrementSensorEventQueueCounter))
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

// push adds a message to the appropriate shard of the queue. Each shard enforces its own
// size cap (see dedupingqueue.WithMaxSize): on overflow the shard evicts its oldest message,
// keeping the most recent ones -- matching the alert manager, which retains the most recent
// violations per alert. For the alerts queue this bounds Central's heap when Sensor produces
// alerts faster than the pipeline drains them.
func (w *workerQueue) push(msg *central.MsgFromSensor) {
	// The zeroth index is reserved for objects that do not match the if statement below
	// w.indexFromKey returns (hashed value % poolSize) + 1 so it cannot return a 0 index
	var idx int
	if msg.GetHashKey() != "" {
		idx = w.indexFromKey(msg.GetHashKey())
	}
	w.queues[idx].Push(msg)
}

func (w *workerQueue) runWorker(ctx context.Context, idx int, stopSig *concurrency.ErrorSignal, deduper hashManager.Deduper, handler func(context.Context, *central.MsgFromSensor) error) {
	queue := w.queues[idx]
	for msg := queue.PullBlocking(stopSig); msg != nil; msg = queue.PullBlocking(stopSig) {
		msgFromSensor, ok := msg.(*central.MsgFromSensor)
		if !ok {
			log.Error("Invalid sensor message")
			continue
		}
		err := handler(ctx, msgFromSensor)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				if pgutils.IsTransientError(err) {
					msgFromSensor.ProcessingAttempt++

					if msgFromSensor.GetProcessingAttempt() == maxHandlerAttempts {
						log.Errorf("Error handling sensor message %T permanently: %v", msgFromSensor.GetEvent().GetResource(), err)
						continue
					}
					reprocessingDuration := time.Duration(msgFromSensor.GetProcessingAttempt()) * handlerRetryInterval
					log.Warnf("Reprocessing sensor message %T in %d minutes: %v", msgFromSensor.GetEvent().GetResource(), int(reprocessingDuration.Minutes()), err)
					concurrency.AfterFunc(reprocessingDuration, func() {
						w.injector.InjectMessageIntoQueue(msgFromSensor)
					}, stopSig)
					continue
				}
				log.Errorf("Unretryable error found while handling sensor message %T permanently: %v", msgFromSensor.GetEvent().GetResource(), err)
			}
			deduper.RemoveMessage(msgFromSensor)
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
