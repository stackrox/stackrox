package lane

import (
	"container/list"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/safe"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	pubsubErrors "github.com/stackrox/rox/sensor/common/pubsub/errors"
	"github.com/stackrox/rox/sensor/common/pubsub/metrics"
)

// DedupingLane provides event deduplication based on a comparable key.
//
// Architecture:
//   - Publish() adds events to an in-memory dedup queue (linked list + map index for O(1) lookups)
//   - Duplicate keys trigger merge via Mergeable interface or WithMergeFunc option
//   - dequeueToChannel() goroutine pulls from dedup queue and writes to channel
//   - run() goroutine reads from channel and dispatches to consumers (concurrent execution)
//
// Deduplication Semantics:
//   - Events in queue (dedupIndex): True deduplication via merge - only 1 event consumed
//   - Events in-flight (inFlightKeys): Re-queued for later - both events consumed (separated in time)
//   - Metrics distinguish these: "deduped" (merged) vs "requeued" (delayed processing)
//
// Trade-offs:
//   - In-flight re-queuing: Events published during consumption are queued (not merged) to preserve
//     the latest state. This means both the in-flight event and the new event will reach consumers,
//     but separated in time. Perfect deduplication would require serializing consumers, which defeats
//     the purpose of concurrent execution.
//   - FIFO ordering: Not strictly guaranteed due to concurrent signal handling and consumer execution.
//   - Production impact: For the resolver use case (sequential deployment updates), true deduplication
//     is 90-100% effective. Under high-velocity concurrent load, 10-20% of events may be re-queued
//     instead of merged. Central's API is idempotent, so functionally correct.
//
// DedupingConfig configures a deduplicating lane.
type DedupingConfig[K comparable] struct {
	Config[*dedupingLane[K]]
}

// WithDedupingLaneSize sets the buffer size for the deduplicating lane's channel.
func WithDedupingLaneSize[K comparable](size int) pubsub.LaneOption[*dedupingLane[K]] {
	return func(lane *dedupingLane[K]) {
		if size < 0 {
			return
		}
		lane.size = size
	}
}

// WithDedupingLaneConsumer sets a custom consumer factory for the deduplicating lane.
func WithDedupingLaneConsumer[K comparable](consumer pubsub.NewConsumer) pubsub.LaneOption[*dedupingLane[K]] {
	return func(lane *dedupingLane[K]) {
		if consumer == nil {
			panic("cannot configure a 'nil' NewConsumer function")
		}
		lane.newConsumerFn = consumer
	}
}

// WithDedupKeyFunc sets the function used to extract deduplication keys from events.
// If the function returns a zero-value key, the event will bypass deduplication.
func WithDedupKeyFunc[K comparable](fn func(pubsub.Event) K) pubsub.LaneOption[*dedupingLane[K]] {
	return func(lane *dedupingLane[K]) {
		if fn == nil {
			panic("cannot configure a 'nil' dedupKeyFunc")
		}
		lane.dedupKeyFunc = fn
	}
}

// WithMergeFunc sets a custom function to merge duplicate events.
// This is used as a fallback if the event does not implement the Mergeable interface.
// The function receives the new event and the old event; it should modify the new event
// to incorporate state from the old event.
func WithMergeFunc[K comparable](fn func(new, old pubsub.Event)) pubsub.LaneOption[*dedupingLane[K]] {
	return func(lane *dedupingLane[K]) {
		lane.mergeFunc = fn
	}
}

// WithDedupSizeMetric sets a Prometheus gauge to track the dedup queue size.
func WithDedupSizeMetric[K comparable](metric prometheus.Gauge) pubsub.LaneOption[*dedupingLane[K]] {
	return func(lane *dedupingLane[K]) {
		lane.sizeMetric = metric
	}
}

// NewDedupingLane creates a new deduplicating lane configuration.
// K is the type of the deduplication key (must be comparable).
func NewDedupingLane[K comparable](id pubsub.LaneID, opts ...pubsub.LaneOption[*dedupingLane[K]]) *DedupingConfig[K] {
	return &DedupingConfig[K]{
		Config: Config[*dedupingLane[K]]{
			id:          id,
			opts:        opts,
			newConsumer: consumer.NewDefaultConsumer(),
		},
	}
}

// NewLane creates and starts a new deduplicating lane.
func (c *DedupingConfig[K]) NewLane() pubsub.Lane {
	lane := &dedupingLane[K]{
		Lane: Lane{
			id:            c.Config.LaneID(),
			newConsumerFn: c.Config.newConsumer,
			consumers:     make(map[pubsub.Topic][]pubsub.Consumer),
		},
		stopper:      concurrency.NewStopper(),
		dedupQueue:   list.New(),
		dedupIndex:   make(map[K]*list.Element),
		inFlightKeys: make(map[K]bool),
		dedupSignal:  concurrency.NewSignal(),
	}
	for _, opt := range c.Config.opts {
		opt(lane)
	}
	if lane.dedupKeyFunc == nil {
		panic("dedupKeyFunc is required for deduplicating lane")
	}
	lane.ch = safe.NewChannel[pubsub.Event](lane.size, lane.stopper.LowLevel().GetStopRequestSignal())
	lane.wg.Add(2)
	go lane.run()
	go lane.dequeueToChannel()
	return lane
}

type dedupingLane[K comparable] struct {
	Lane
	size    int
	ch      *safe.Channel[pubsub.Event]
	stopper concurrency.Stopper
	// wg tracks completion of the run() and dequeueToChannel() goroutines. The
	// stopper's Stopped() signal is shared by both goroutines and latches on the
	// first ReportStopped() call, so it cannot be used to wait for both to exit.
	wg sync.WaitGroup

	// Dedup state
	dedupLock    sync.Mutex
	dedupQueue   *list.List
	dedupIndex   map[K]*list.Element
	inFlightKeys map[K]bool // Keys currently being consumed (not in queue/index, but still processing)
	dedupKeyFunc func(pubsub.Event) K
	mergeFunc    func(new, old pubsub.Event)
	sizeMetric   prometheus.Gauge
	// Signal that new events are available in dedup queue
	dedupSignal concurrency.Signal
}

// Publish publishes an event to the lane, deduplicating if a duplicate key is found.
func (l *dedupingLane[K]) Publish(event pubsub.Event) error {
	// Check if lane is stopped
	select {
	case <-l.stopper.Flow().StopRequested():
		return errors.Wrap(pubsubErrors.NewPublishOnStoppedLaneErr(l.id), "unable to publish event")
	default:
	}

	topic := event.Topic()

	// Extract dedup key
	key := l.dedupKeyFunc(event)

	l.dedupLock.Lock()
	defer l.dedupLock.Unlock()

	// Check for zero-value key (bypass dedup - add to queue without indexing)
	var zeroKey K
	if key == zeroKey {
		l.dedupQueue.PushBack(event)
		l.updateSizeMetric()
		l.dedupSignal.Signal()
		metrics.RecordPublishOperation(l.id, topic, metrics.Published)
		return nil
	}

	// Check for duplicate in queue
	if oldElem, exists := l.dedupIndex[key]; exists {
		oldEvent := oldElem.Value.(pubsub.Event)

		// Try interface-based merge first
		if mergeable, ok := event.(pubsub.Mergeable[K]); ok {
			mergeable.MergeFrom(oldEvent)
		} else if l.mergeFunc != nil {
			// Fallback to function-based merge
			l.mergeFunc(event, oldEvent)
		}
		// else: last-write-wins (replace)

		// Replace old event with new (maintains position in queue)
		oldElem.Value = event

		metrics.RecordPublishOperation(l.id, topic, metrics.Deduped)
		l.updateSizeMetric()
		// No signal needed - event is already in queue, just updated
		return nil
	}

	// Check if key is in-flight (being consumed)
	if l.inFlightKeys[key] {
		// Event is currently being consumed. Add new event to queue.
		// When consumption completes, this will be processed next.
		// Note: This is not true deduplication - both events will be consumed
		// (separated in time). We use "Requeued" to distinguish from merge-based dedup.
		elem := l.dedupQueue.PushBack(event)
		l.dedupIndex[key] = elem
		metrics.RecordPublishOperation(l.id, topic, metrics.Requeued)
		l.updateSizeMetric()
		l.dedupSignal.Signal()
		return nil
	}

	// New key: add to queue and index
	elem := l.dedupQueue.PushBack(event)
	l.dedupIndex[key] = elem

	metrics.RecordPublishOperation(l.id, topic, metrics.Published)
	l.updateSizeMetric()
	l.dedupSignal.Signal()
	return nil
}

func (l *dedupingLane[K]) updateSizeMetric() {
	if l.sizeMetric != nil {
		l.sizeMetric.Set(float64(l.dedupQueue.Len()))
	}
}

// dequeueToChannel pulls events from the dedup queue and writes them to the channel.
// This goroutine ensures that only merged/latest events reach the channel.
func (l *dedupingLane[K]) dequeueToChannel() {
	defer l.wg.Done()
	defer l.stopper.Flow().ReportStopped()
	for {
		// Wait for signal or stop
		select {
		case <-l.stopper.Flow().StopRequested():
			return
		case <-l.dedupSignal.Done():
		}

		// Process all queued events (signal may represent multiple publishes)
		for {
			// Pull front of queue - use closure to ensure deferred unlock
			event, shouldBreak := func() (pubsub.Event, bool) {
				l.dedupLock.Lock()
				defer l.dedupLock.Unlock()

				if l.dedupQueue.Len() == 0 {
					// Queue empty - reset signal and wait for next publish
					l.dedupSignal.Reset()
					return nil, true
				}
				front := l.dedupQueue.Front()
				event := front.Value.(pubsub.Event)

				// Remove from queue and index, mark as in-flight
				l.dedupQueue.Remove(front)
				key := l.dedupKeyFunc(event)
				var zeroKey K
				if key != zeroKey {
					delete(l.dedupIndex, key)
					l.inFlightKeys[key] = true
				}
				l.updateSizeMetric()
				return event, false
			}()

			if shouldBreak {
				break
			}

			// Write to channel (may block if channel is full)
			select {
			case <-l.stopper.Flow().StopRequested():
				return
			default:
				if err := l.ch.Write(event); err != nil {
					// Channel write failed (lane stopped) - exit
					return
				}
				metrics.SetQueueSize(l.id, l.ch.Len())
			}
		}
	}
}

func (l *dedupingLane[K]) run() {
	defer l.wg.Done()
	defer l.stopper.Flow().ReportStopped()
	for {
		// Priority 1: Check if stop requested
		select {
		case <-l.stopper.Flow().StopRequested():
			return
		default:
		}
		// Priority 2: Read event, but respect stop during blocking read
		select {
		case <-l.stopper.Flow().StopRequested():
			return
		case event, ok := <-l.ch.Chan():
			if !ok {
				return
			}
			l.handleEvent(event)
		}
	}
}

func (l *dedupingLane[K]) handleEvent(event pubsub.Event) {
	defer metrics.SetQueueSize(l.id, l.ch.Len())

	// Clear in-flight status after all consumers complete
	key := l.dedupKeyFunc(event)
	var zeroKey K
	if key != zeroKey {
		defer func() {
			l.dedupLock.Lock()
			defer l.dedupLock.Unlock()
			delete(l.inFlightKeys, key)
		}()
	}

	// Handle event (same as ConcurrentLane)
	consumers, err := l.getConsumersByTopic(event.Topic())
	if err != nil {
		rateLimitedLog.ErrorL(l.id.String(), "unable to handle event: %v", err)
		metrics.RecordConsumerOperation(l.id, event.Topic(), pubsub.NoConsumers, metrics.NoConsumers)
		return
	}

	// Wait for all consumers to complete before clearing in-flight status
	var wg sync.WaitGroup
	for _, c := range consumers {
		wg.Add(1)
		errC := c.Consume(l.stopper.Client().Stopped(), event)
		go func(errC <-chan error) {
			defer wg.Done()
			l.handleConsumerError(errC)
		}(errC)
	}
	wg.Wait()
}

func (l *dedupingLane[K]) getConsumersByTopic(topic pubsub.Topic) ([]pubsub.Consumer, error) {
	l.consumerLock.RLock()
	defer l.consumerLock.RUnlock()
	consumers, ok := l.consumers[topic]
	if !ok {
		return nil, errors.Wrap(pubsubErrors.NewConsumersNotFoundForTopicErr(topic, l.id), "unable to handle event")
	}
	return consumers, nil
}

func (l *dedupingLane[K]) handleConsumerError(errC <-chan error) {
	select {
	case err := <-errC:
		if err != nil {
			rateLimitedLog.ErrorL(l.id.String(), "unable to handle event: %v", err)
		}
	case <-l.stopper.Flow().StopRequested():
	}
}

func (l *dedupingLane[K]) RegisterConsumer(consumerID pubsub.ConsumerID, topic pubsub.Topic, callback pubsub.EventCallback) error {
	if callback == nil {
		return errors.New("cannot register a 'nil' callback")
	}
	c, err := l.newConsumerFn(l.id, topic, consumerID, callback)
	if err != nil {
		return errors.Wrap(err, "creating the consumer")
	}
	l.consumerLock.Lock()
	defer l.consumerLock.Unlock()
	l.consumers[topic] = append(l.consumers[topic], c)
	return nil
}

func (l *dedupingLane[K]) Stop() {
	l.stopper.Client().Stop()
	// Wait for both run() and dequeueToChannel() goroutines to fully exit.
	// The channel will be closed automatically when the stop request signal triggers.
	l.wg.Wait()
	l.Lane.Stop()
}
