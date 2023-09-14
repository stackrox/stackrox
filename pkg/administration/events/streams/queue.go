package streams

import (
	"container/list"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	// Sample calculation with a sample administration event (250 chars in message + hint):
	// 1 Administration event = 160 bytes
	// 100000 *160 bytes = 16 MB
	maxQueueSize = 100000
)

var (
	administrationEventsQueueCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      "administration_events_queue",
		Help:      "A counter that tracks the size of the administration events queue",
	}, []string{"Operation"})
)

type administrationEventsQueue struct {
	mutex       sync.Mutex
	queue       *list.List
	notEmptySig concurrency.Signal
}

func newQueue() *administrationEventsQueue {
	return &administrationEventsQueue{
		notEmptySig: concurrency.NewSignal(),
		queue:       list.New(),
	}
}

func (q *administrationEventsQueue) pullBlocking(waitable concurrency.Waitable) *events.AdministrationEvent {
	var event *events.AdministrationEvent
	for event == nil {
		select {
		case <-waitable.Done():
			return nil
		case <-q.notEmptySig.Done():
			event = q.pull()
		}
	}
	return event
}

func (q *administrationEventsQueue) pull() *events.AdministrationEvent {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.queue.Len() == 0 {
		return nil
	}

	event := q.queue.Remove(q.queue.Front()).(*events.AdministrationEvent)
	administrationEventsQueueCounter.With(prometheus.Labels{"Operation": metrics.Remove.String()}).Inc()

	if q.queue.Len() == 0 {
		q.notEmptySig.Reset()
	}
	return event
}

func (q *administrationEventsQueue) push(event *events.AdministrationEvent) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.queue.Len() >= maxQueueSize {
		return
	}

	defer q.notEmptySig.Signal()
	administrationEventsQueueCounter.With(prometheus.Labels{"Operation": metrics.Add.String()}).Inc()
	q.queue.PushBack(event)
}
