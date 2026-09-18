package connection

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/backgroundworker"
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

	activeWorkers atomic.Int32
	statusAdapter *backgroundworker.StatusAdapter
}

func newWorkerQueue(poolSize int, typ string, injector common.MessageInjector) *workerQueue {
	totalSize := poolSize + 1
	queues := make([]*dedupingqueue.DedupingQueue[string], totalSize)
	for i := range totalSize {
		queues[i] = dedupingqueue.NewDedupingQueue[string](
			dedupingqueue.WithQueueName[string](typ),
			dedupingqueue.WithOperationMetricsFunc[string](metrics.IncrementSensorEventQueueCounter))
	}

	w := &workerQueue{
		poolSize:  poolSize,
		totalSize: totalSize,
		queues:    queues,
		injector:  injector,
	}

	// Register with the background worker registry for debug endpoint
	// observability. workerQueue does not fit a standard archetype: it is a
	// sharded pool of per-type consumers with custom retry/backoff logic,
	// created and torn down per sensor connection. Deregistered in run()
	// once all shard workers have exited.
	w.statusAdapter = &backgroundworker.StatusAdapter{
		WorkerName: fmt.Sprintf("sensor-event-queue-%s-%p", typ, w),
		WorkerKind: "pipeline",
		StatusFunc: func() backgroundworker.WorkerStatus {
			depths := make([]int, len(queues))
			total := 0
			for i, q := range queues {
				depths[i] = q.Len()
				total += depths[i]
			}
			return backgroundworker.WorkerStatus{
				State: "running",
				Extra: map[string]any{
					"event_type":        typ,
					"shard_count":       poolSize,
					"active_workers":    w.activeWorkers.Load(),
					"queue_depths":      depths,
					"queue_depth_total": total,
				},
			}
		},
	}
	backgroundworker.Global.Register(w.statusAdapter)

	return w
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
	// The zeroth index is reserved for objects that do not match the if statement below
	// w.indexFromKey returns (hashed value % poolSize) + 1 so it cannot return a 0 index
	var idx int
	if msg.GetHashKey() != "" {
		idx = w.indexFromKey(msg.GetHashKey())
	}

	w.queues[idx].Push(msg)
}

func (w *workerQueue) runWorker(ctx context.Context, idx int, stopSig *concurrency.ErrorSignal, deduper hashManager.Deduper, handler func(context.Context, *central.MsgFromSensor) error) {
	w.activeWorkers.Add(1)
	defer w.activeWorkers.Add(-1)

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
	backgroundworker.Global.Deregister(w.statusAdapter)
}
