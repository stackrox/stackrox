// Package rate provides workload-level fair rate limiting across clients.
// It manages per-client token buckets using golang.org/x/time/rate.Limiter.
// In this package, Limiter is the higher-level manager that allocates and
// rebalances per-client limiters; the underlying *rate.Limiter values are the
// per-client buckets used for enforcement.
package rate

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	gorate "golang.org/x/time/rate"
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

// Clock provides an abstraction over time for testing.
type Clock interface {
	Now() time.Time
}

// Limiter provides per-client fair rate limiting for any workload type.
// Each client gets an equal share (1/N) of the global capacity, with automatic
// rebalancing when clients connect or disconnect.
type Limiter struct {
	workloadName   string  // name for logging/metrics (e.g., "vm_index_report")
	globalRate     float64 // requests per second (0 = unlimited)
	bucketCapacity int     // max tokens per client bucket (allows temporary bursts)

	mu         sync.RWMutex
	buckets    map[string]*gorate.Limiter
	numClients int

	clock Clock // time source (injectable for testing)
}

// RealClock uses the system clock.
type RealClock struct{}

// Now returns the current time.
func (RealClock) Now() time.Time {
	return time.Now()
}

// NewLimiter creates a new per-client rate limiter for the given workload.
// globalRate of 0 disables rate limiting (unlimited).
// bucketCapacity is the max tokens per client bucket (allows temporary bursts above sustained rate).
// workloadName is used for logging and metrics (e.g., "vm_index_report", "node_inventory").
//
// Returns error if:
//   - workloadName is empty
//   - globalRate is negative
//   - bucketCapacity is less than 1
func NewLimiter(workloadName string, globalRate float64, bucketCapacity int) (*Limiter, error) {
	return NewLimiterWithClock(workloadName, globalRate, bucketCapacity, RealClock{})
}

// NewLimiterWithClock creates a new per-client rate limiter with an injectable clock.
// This is primarily useful for testing to control time and avoid flaky tests.
// For production use, prefer NewLimiter which uses the real system clock.
func NewLimiterWithClock(workloadName string, globalRate float64, bucketCapacity int, clock Clock) (*Limiter, error) {
	if workloadName == "" {
		return nil, ErrEmptyWorkloadName
	}
	if globalRate < 0 {
		return nil, ErrNegativeRate
	}
	if bucketCapacity < 1 {
		return nil, ErrInvalidBucketCapacity
	}

	// Initialize metrics for this workload so they're visible in Prometheus immediately.
	// Set per-client metrics to the maximum values (what a single client would get).
	RequestsTotal.WithLabelValues(workloadName, OutcomeAccepted).Add(0)
	RequestsTotal.WithLabelValues(workloadName, OutcomeRejected).Add(0)
	ActiveClients.WithLabelValues(workloadName).Set(0)
	PerClientRate.WithLabelValues(workloadName).Set(globalRate)
	PerClientBucketCapacity.WithLabelValues(workloadName).Set(float64(bucketCapacity))

	return &Limiter{
		workloadName:   workloadName,
		globalRate:     globalRate,
		bucketCapacity: bucketCapacity,
		buckets:        make(map[string]*gorate.Limiter),
		clock:          clock,
	}, nil
}

// TryConsume attempts to consume one token for the given client.
// Returns true if allowed, false if rate limit exceeded.
// Metrics are automatically recorded.
func (l *Limiter) TryConsume(clientID string) (allowed bool, reason string) {
	if l.globalRate <= 0 {
		// Rate limiting disabled, but still record metrics for visibility into request volume.
		RequestsTotal.WithLabelValues(l.workloadName, OutcomeAccepted).Inc()
		return true, ""
	}

	limiter := l.getOrCreateLimiter(clientID)

	// Use AllowN with clock.Now() to support time injection for testing.
	if limiter.AllowN(l.clock.Now(), 1) {
		RequestsTotal.WithLabelValues(l.workloadName, OutcomeAccepted).Inc()
		return true, ""
	}

	RequestsTotal.WithLabelValues(l.workloadName, OutcomeRejected).Inc()
	return false, ReasonRateLimitExceeded
}

// getOrCreateLimiter returns the rate limiter for a given client, creating one if needed.
// When a new client is added, all limiters are rebalanced to maintain fairness.
func (l *Limiter) getOrCreateLimiter(clientID string) *gorate.Limiter {
	// Fast path: read lock to check if limiter exists
	if limiter := concurrency.WithRLock1(&l.mu, func() *gorate.Limiter {
		return l.buckets[clientID]
	}); limiter != nil {
		return limiter
	}

	// Slow path: write lock to create new limiter
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock to avoid race conditions.
	if limiter, ok := l.buckets[clientID]; ok {
		return limiter
	}

	if clientID == "" {
		log.Warnf("getOrCreateLimiter called with empty clientID for workload %s; this may cause unfair rate limiting", l.workloadName)
	}

	l.numClients++
	perClientRate := l.globalRate / float64(l.numClients)
	bucketCapacity := l.perClientBucketCapacity(l.numClients)

	// Create limiter for this client
	newLimiter := gorate.NewLimiter(gorate.Limit(perClientRate), bucketCapacity)
	l.buckets[clientID] = newLimiter

	// Rebalance all limiters (including the new one) and update metrics
	for _, limiter := range l.buckets {
		limiter.SetLimit(gorate.Limit(perClientRate))
		limiter.SetBurst(bucketCapacity)
	}

	ActiveClients.WithLabelValues(l.workloadName).Set(float64(l.numClients))
	PerClientRate.WithLabelValues(l.workloadName).Set(perClientRate)
	PerClientBucketCapacity.WithLabelValues(l.workloadName).Set(float64(bucketCapacity))

	log.Debugf("New client %s registered for %s rate limiting (clients: %d, rate: %.2f req/s, max bucket capacity: %d)",
		clientID, l.workloadName, l.numClients, perClientRate, bucketCapacity)

	return newLimiter
}

// perClientBucketCapacity calculates the per-client burst capacity (max tokens in bucket).
// The global bucket capacity is divided equally among all clients.
func (l *Limiter) perClientBucketCapacity(numClients int) int {
	burst := l.bucketCapacity / numClients
	if burst < 1 {
		burst = 1
	}
	return burst
}

// OnClientDisconnect removes a client from rate limiting and rebalances remaining limiters.
// This should be called when a client connection is terminated.
func (l *Limiter) OnClientDisconnect(clientID string) {
	if l.globalRate <= 0 {
		// Rate limiting disabled
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.buckets[clientID]; !ok {
		return
	}

	delete(l.buckets, clientID)
	l.numClients--

	log.Infof("Client %s disconnected from %s rate limiting (remaining clients: %d)",
		clientID, l.workloadName, l.numClients)

	if l.numClients == 0 {
		ActiveClients.WithLabelValues(l.workloadName).Set(0)
		PerClientRate.WithLabelValues(l.workloadName).Set(l.globalRate)
		PerClientBucketCapacity.WithLabelValues(l.workloadName).Set(float64(l.bucketCapacity))
		return
	}

	perClientRate := l.globalRate / float64(l.numClients)
	bucketCapacity := l.perClientBucketCapacity(l.numClients)

	for _, limiter := range l.buckets {
		limiter.SetLimit(gorate.Limit(perClientRate))
		limiter.SetBurst(bucketCapacity)
	}

	ActiveClients.WithLabelValues(l.workloadName).Set(float64(l.numClients))
	PerClientRate.WithLabelValues(l.workloadName).Set(perClientRate)
	PerClientBucketCapacity.WithLabelValues(l.workloadName).Set(float64(bucketCapacity))
}

// numActiveClients returns the number of currently active clients.
func (l *Limiter) numActiveClients() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.numClients
}

// GlobalRate returns the configured global rate limit.
func (l *Limiter) GlobalRate() float64 {
	return l.globalRate
}

// BucketCapacity returns the configured bucket capacity.
func (l *Limiter) BucketCapacity() int {
	return l.bucketCapacity
}

// WorkloadName returns the workload name for this limiter.
func (l *Limiter) WorkloadName() string {
	return l.workloadName
}
