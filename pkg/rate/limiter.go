// Package rate provides workload-level fair concurrency limiting across clients.
// It manages per-client token counters that are decremented on consumption and
// incremented when processing completes.  Tokens are never refilled by time;
// they are returned explicitly after a unit of work finishes.
package rate

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	// ErrEmptyWorkloadName is returned when workloadName is empty.
	ErrEmptyWorkloadName = errors.New("workloadName must not be empty")
	// ErrNegativeRate is returned when globalRate is negative.
	ErrNegativeRate = errors.New("globalRate must be >= 0")
	// ErrInvalidBucketCapacity is returned when bucketCapacity is less than 1.
	ErrInvalidBucketCapacity = errors.New("bucketCapacity must be >= 1")
)

const (
	// ReasonRateLimitExceeded is the reason returned when a request is rejected due to rate limiting.
	ReasonRateLimitExceeded = "rate limit exceeded"
)

// clientBucket tracks the available and maximum token count for a single client.
type clientBucket struct {
	available int // tokens currently available for consumption
	capacity  int // maximum tokens this client may hold
}

// Limiter provides per-client fair concurrency limiting for any workload type.
// Each client gets an equal share (1/N) of the global capacity, with automatic
// rebalancing when clients connect or disconnect.
//
// Unlike a traditional rate limiter, tokens are not refilled over time. Instead,
// tokens are returned explicitly via Return when processing of a previously
// consumed token completes. This ensures that capacity tracks actual processing
// throughput: fast processing yields high throughput, slow processing yields
// natural backpressure.
type Limiter struct {
	workloadName   string
	globalRate     float64 // > 0 enables limiting, 0 disables (unlimited)
	bucketCapacity int     // total tokens shared across all clients

	mu         sync.Mutex
	buckets    map[string]*clientBucket
	numClients int

	acceptsFn func(msg *central.MsgFromSensor) bool
}

// limiterOption is an intermediary type for creating a limiter.
// It forces the caller to define the workload type to which the limiter reacts.
type limiterOption struct {
	l   *Limiter
	err error
}

// ForAllWorkloads configures the limiter to analyze all types of messages (including nil).
func (lo *limiterOption) ForAllWorkloads() (*Limiter, error) {
	if lo.l == nil {
		return nil, lo.err
	}
	lo.l.acceptsFn = func(msg *central.MsgFromSensor) bool {
		return true
	}
	return lo.l, lo.err
}

// ForWorkload allows specifying a function that should return true if the given
// MsgFromSensor is to be evaluated by the limiter (and later accepted or
// rejected) and false if the limiter should ignore this message (allowing it to
// pass). This function must handle nil arguments and execute quickly.
func (lo *limiterOption) ForWorkload(acceptsFn func(msg *central.MsgFromSensor) bool) (*Limiter, error) {
	if lo.l == nil {
		return nil, lo.err
	}
	if acceptsFn == nil {
		return nil, errors.New("acceptsFn must not be nil")
	}
	lo.l.acceptsFn = acceptsFn
	return lo.l, lo.err
}

// NewLimiter creates a new per-client concurrency limiter for the given workload.
// globalRate > 0 enables limiting; globalRate of 0 disables it (unlimited).
// bucketCapacity is the total number of tokens shared across all clients.
// workloadName is used for logging and metrics.
//
// Returns error if:
//   - workloadName is empty
//   - globalRate is negative
//   - bucketCapacity is less than 1
func NewLimiter(workloadName string, globalRate float64, bucketCapacity int) *limiterOption {
	if workloadName == "" {
		return &limiterOption{nil, ErrEmptyWorkloadName}
	}
	if globalRate < 0 {
		return &limiterOption{nil, ErrNegativeRate}
	}
	if bucketCapacity < 1 {
		return &limiterOption{nil, ErrInvalidBucketCapacity}
	}

	// Initialize metrics so they are visible in Prometheus immediately.
	RequestsTotal.WithLabelValues(workloadName, OutcomeAccepted).Add(0)
	RequestsTotal.WithLabelValues(workloadName, OutcomeRejected).Add(0)
	ActiveClients.WithLabelValues(workloadName).Set(0)
	InFlightTokens.WithLabelValues(workloadName).Set(0)
	PerClientBucketCapacity.WithLabelValues(workloadName).Set(float64(bucketCapacity))

	return &limiterOption{
		l: &Limiter{
			workloadName:   workloadName,
			globalRate:     globalRate,
			bucketCapacity: bucketCapacity,
			buckets:        make(map[string]*clientBucket),
		},
		err: nil,
	}
}

// TryConsume attempts to consume one token for the given client.
// Returns true if allowed, false if no tokens are available.
// Metrics are automatically recorded.
//
// Every successful TryConsume must eventually be paired with a Return call
// when the associated work completes; failing to do so permanently reduces
// available capacity.
func (l *Limiter) TryConsume(clientID string, msg *central.MsgFromSensor) (allowed bool, reason string) {
	if l == nil {
		return true, "nil rate limiter"
	}
	if !l.accepts(msg) {
		return true, ""
	}

	if l.globalRate <= 0 {
		RequestsTotal.WithLabelValues(l.workloadName, OutcomeAccepted).Inc()
		return true, ""
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.getOrCreateBucketLocked(clientID)
	if bucket.available > 0 {
		bucket.available--
		RequestsTotal.WithLabelValues(l.workloadName, OutcomeAccepted).Inc()
		InFlightTokens.WithLabelValues(l.workloadName).Inc()
		return true, ""
	}

	RequestsTotal.WithLabelValues(l.workloadName, OutcomeRejected).Inc()
	return false, ReasonRateLimitExceeded
}

// Return returns one token for the given client after processing completes.
// If the message does not match the workload filter, or if limiting is
// disabled, Return is a no-op. It is safe to call Return for a client that
// has already disconnected; the token is silently discarded.
func (l *Limiter) Return(clientID string, msg *central.MsgFromSensor) {
	if l == nil {
		return
	}
	if !l.accepts(msg) {
		return
	}
	if l.globalRate <= 0 {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[clientID]
	if !ok {
		log.Debugf("Token return for disconnected client %s on %s (token discarded)",
			clientID, l.workloadName)
		return
	}
	if bucket.available < bucket.capacity {
		bucket.available++
		InFlightTokens.WithLabelValues(l.workloadName).Dec()
	}
}

func (l *Limiter) accepts(msg *central.MsgFromSensor) bool {
	if l.acceptsFn == nil {
		return true
	}
	return l.acceptsFn(msg)
}

// getOrCreateBucketLocked returns the bucket for a given client, creating one
// if needed.  When a new client is added, all buckets are rebalanced to
// maintain fairness.  Must be called with l.mu held.
func (l *Limiter) getOrCreateBucketLocked(clientID string) *clientBucket {
	if bucket, ok := l.buckets[clientID]; ok {
		return bucket
	}

	if clientID == "" {
		log.Warnf("getOrCreateBucketLocked called with empty clientID for workload %s; this may cause unfair limiting", l.workloadName)
	}

	l.numClients++
	capacity := l.perClientBucketCapacity(l.numClients)

	newBucket := &clientBucket{
		available: capacity,
		capacity:  capacity,
	}
	l.buckets[clientID] = newBucket

	// Rebalance all buckets (including the new one) to the new capacity.
	for _, bucket := range l.buckets {
		bucket.capacity = capacity
		if bucket.available > capacity {
			bucket.available = capacity
		}
	}

	ActiveClients.WithLabelValues(l.workloadName).Set(float64(l.numClients))
	PerClientBucketCapacity.WithLabelValues(l.workloadName).Set(float64(capacity))

	log.Debugf("New client %s registered for %s concurrency limiting (clients: %d, per-client capacity: %d)",
		clientID, l.workloadName, l.numClients, capacity)

	return newBucket
}

// perClientBucketCapacity calculates the per-client token capacity.
// The global bucket capacity is divided equally among all clients.
func (l *Limiter) perClientBucketCapacity(numClients int) int {
	burst := l.bucketCapacity / numClients
	if burst < 1 {
		burst = 1
	}
	return burst
}

// OnClientDisconnect removes a client from the limiter and rebalances
// remaining clients.  In-flight tokens for the disconnected client are
// discarded (the corresponding InFlightTokens gauge is decremented).
func (l *Limiter) OnClientDisconnect(clientID string) {
	if l == nil {
		return
	}
	if l.globalRate <= 0 {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[clientID]
	if !ok {
		return
	}

	inFlight := bucket.capacity - bucket.available
	if inFlight > 0 {
		InFlightTokens.WithLabelValues(l.workloadName).Sub(float64(inFlight))
	}

	delete(l.buckets, clientID)
	l.numClients--

	log.Infof("Client %s disconnected from %s concurrency limiting (remaining clients: %d)",
		clientID, l.workloadName, l.numClients)

	if l.numClients == 0 {
		ActiveClients.WithLabelValues(l.workloadName).Set(0)
		PerClientBucketCapacity.WithLabelValues(l.workloadName).Set(float64(l.bucketCapacity))
		return
	}

	capacity := l.perClientBucketCapacity(l.numClients)
	for _, bucket := range l.buckets {
		bucket.capacity = capacity
		if bucket.available > capacity {
			bucket.available = capacity
		}
	}

	ActiveClients.WithLabelValues(l.workloadName).Set(float64(l.numClients))
	PerClientBucketCapacity.WithLabelValues(l.workloadName).Set(float64(capacity))
}

// numActiveClients returns the number of currently active clients.
func (l *Limiter) numActiveClients() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.numClients
}

// GlobalRate returns the configured global rate (used as an enabled flag).
func (l *Limiter) GlobalRate() float64 {
	return l.globalRate
}

// BucketCapacity returns the configured total bucket capacity.
func (l *Limiter) BucketCapacity() int {
	return l.bucketCapacity
}

// WorkloadName returns the workload name for this limiter.
func (l *Limiter) WorkloadName() string {
	return l.workloadName
}
