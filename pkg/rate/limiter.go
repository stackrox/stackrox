package rate

import (
	"time"

	"github.com/pkg/errors"
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

	buckets sync.Map // map[clientID]*gorate.Limiter
	clock   Clock    // time source (injectable for testing)
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
	return &Limiter{
		workloadName:   workloadName,
		globalRate:     globalRate,
		bucketCapacity: bucketCapacity,
		clock:          clock,
	}, nil
}

// TryConsume attempts to consume one token for the given client.
// Returns true if allowed, false if rate limit exceeded.
// Metrics are automatically recorded.
func (l *Limiter) TryConsume(clientID string) (allowed bool, reason string) {
	if l.globalRate <= 0 {
		// Rate limiting disabled
		return true, ""
	}

	limiter := l.getOrCreateLimiter(clientID)

	// Use AllowN with clock.Now() to support time injection for testing.
	if limiter.AllowN(l.clock.Now(), 1) {
		RequestsTotal.WithLabelValues(l.workloadName, OutcomeAccepted).Inc()
		RequestsAccepted.WithLabelValues(l.workloadName, clientID).Inc()
		return true, ""
	}

	RequestsTotal.WithLabelValues(l.workloadName, OutcomeRejected).Inc()
	RequestsRejected.WithLabelValues(l.workloadName, clientID, ReasonRateLimitExceeded).Inc()
	return false, ReasonRateLimitExceeded
}

// getOrCreateLimiter returns the rate limiter for a given client, creating one if needed.
// When a new client is added, all limiters are rebalanced to maintain fairness.
func (l *Limiter) getOrCreateLimiter(clientID string) *gorate.Limiter {
	if val, ok := l.buckets.Load(clientID); ok {
		return val.(*gorate.Limiter)
	}

	// New client - create limiter and rebalance all
	numClients := l.countActiveClients() + 1 // +1 for the new one
	perClientRate := l.globalRate / float64(numClients)
	bucketCapacity := l.perClientBucketCapacity(numClients)

	limiter := gorate.NewLimiter(gorate.Limit(perClientRate), bucketCapacity)
	l.buckets.Store(clientID, limiter)

	log.Infof("New client %s registered for %s rate limiting (clients: %d, rate: %.2f req/s, max bucket capacity: %d)",
		clientID, l.workloadName, numClients, perClientRate, bucketCapacity)

	// Rebalance all existing limiters with new client count
	l.rebalanceLimiters()

	return limiter
}

// countActiveClients returns the number of currently active clients.
func (l *Limiter) countActiveClients() int {
	count := 0
	l.buckets.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
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

// rebalanceLimiters updates all existing limiters with the new per-client rate.
// This is called when a new client connects to maintain fairness.
func (l *Limiter) rebalanceLimiters() {
	numClients := l.countActiveClients()
	if numClients == 0 {
		PerClientRate.WithLabelValues(l.workloadName).Set(0)
		PerClientBucketCapacity.WithLabelValues(l.workloadName).Set(0)
		return
	}

	perClientRate := l.globalRate / float64(numClients)
	bucketCapacity := l.perClientBucketCapacity(numClients)

	l.buckets.Range(func(key, val interface{}) bool {
		limiter := val.(*gorate.Limiter)
		limiter.SetLimit(gorate.Limit(perClientRate))
		limiter.SetBurst(bucketCapacity)
		return true
	})

	PerClientRate.WithLabelValues(l.workloadName).Set(perClientRate)
	PerClientBucketCapacity.WithLabelValues(l.workloadName).Set(float64(bucketCapacity))

	log.Debugf("Rebalanced %s rate limiters: %d clients, %.2f req/s each, burst %d",
		l.workloadName, numClients, perClientRate, bucketCapacity)
}

// OnClientDisconnect removes a client from rate limiting and rebalances remaining limiters.
// This should be called when a client connection is terminated.
func (l *Limiter) OnClientDisconnect(clientID string) {
	if l.globalRate <= 0 {
		// Rate limiting disabled
		return
	}

	// Check if this client was tracked
	if _, ok := l.buckets.Load(clientID); !ok {
		return
	}

	l.buckets.Delete(clientID)

	numClients := l.countActiveClients()
	log.Infof("Client %s disconnected from %s rate limiting (remaining clients: %d)",
		clientID, l.workloadName, numClients)

	// Rebalance remaining limiters to give them more capacity
	l.rebalanceLimiters()
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
