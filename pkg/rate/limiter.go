package rate

import (
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

// Limiter provides per-sensor fair rate limiting for any workload type.
// Each sensor gets an equal share (1/N) of the global capacity, with automatic
// rebalancing when sensors connect or disconnect.
type Limiter struct {
	workloadName   string  // name for logging/metrics (e.g., "vm_index_report")
	globalRate     float64 // requests per second (0 = unlimited)
	bucketCapacity int     // max tokens per sensor bucket (allows temporary bursts)

	buckets sync.Map // map[sensorID]*gorate.Limiter
}

// NewLimiter creates a new per-sensor rate limiter for the given workload.
// globalRate of 0 disables rate limiting (unlimited).
// bucketCapacity is the max tokens per sensor bucket (allows temporary bursts above sustained rate).
// workloadName is used for logging and metrics (e.g., "vm_index_report", "node_inventory").
//
// Returns error if:
//   - workloadName is empty
//   - globalRate is negative
//   - bucketCapacity is less than 1
func NewLimiter(workloadName string, globalRate float64, bucketCapacity int) (*Limiter, error) {
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
	}, nil
}

// TryConsume attempts to consume one token for the given sensor.
// Returns true if allowed, false if rate limit exceeded.
// Metrics are automatically recorded.
func (l *Limiter) TryConsume(sensorID string) (allowed bool, reason string) {
	if l.globalRate <= 0 {
		// Rate limiting disabled
		return true, ""
	}

	limiter := l.getOrCreateLimiter(sensorID)

	if limiter.Allow() {
		RequestsTotal.WithLabelValues(l.workloadName, OutcomeAccepted).Inc()
		RequestsAccepted.WithLabelValues(l.workloadName, sensorID).Inc()
		return true, ""
	}

	RequestsTotal.WithLabelValues(l.workloadName, OutcomeRejected).Inc()
	RequestsRejected.WithLabelValues(l.workloadName, sensorID, ReasonRateLimitExceeded).Inc()
	return false, ReasonRateLimitExceeded
}

// getOrCreateLimiter returns the rate limiter for a given sensor, creating one if needed.
// When a new sensor is added, all limiters are rebalanced to maintain fairness.
func (l *Limiter) getOrCreateLimiter(sensorID string) *gorate.Limiter {
	if val, ok := l.buckets.Load(sensorID); ok {
		return val.(*gorate.Limiter)
	}

	// New sensor - create limiter and rebalance all
	numSensors := l.countActiveSensors() + 1 // +1 for the new one
	perSensorRate := l.globalRate / float64(numSensors)
	bucketCapacity := l.perSensorBucketCapacity(numSensors)

	limiter := gorate.NewLimiter(gorate.Limit(perSensorRate), bucketCapacity)
	l.buckets.Store(sensorID, limiter)

	log.Infof("New sensor %s registered for %s rate limiting (sensors: %d, rate: %.2f req/s, max bucket capacity: %d)",
		sensorID, l.workloadName, numSensors, perSensorRate, bucketCapacity)

	// Rebalance all existing limiters with new sensor count
	l.rebalanceLimiters()

	return limiter
}

// countActiveSensors returns the number of currently active sensors.
func (l *Limiter) countActiveSensors() int {
	count := 0
	l.buckets.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// perSensorBucketCapacity calculates the per-sensor burst capacity (max tokens in bucket).
// The global bucket capacity is divided equally among all sensors.
func (l *Limiter) perSensorBucketCapacity(numSensors int) int {
	burst := l.bucketCapacity / numSensors
	if burst < 1 {
		burst = 1
	}
	return burst
}

// rebalanceLimiters updates all existing limiters with the new per-sensor rate.
// This is called when a new sensor connects to maintain fairness.
func (l *Limiter) rebalanceLimiters() {
	numSensors := l.countActiveSensors()
	if numSensors == 0 {
		PerSensorRate.WithLabelValues(l.workloadName).Set(0)
		PerSensorBucketCapacity.WithLabelValues(l.workloadName).Set(0)
		return
	}

	perSensorRate := l.globalRate / float64(numSensors)
	bucketCapacity := l.perSensorBucketCapacity(numSensors)

	l.buckets.Range(func(key, val interface{}) bool {
		limiter := val.(*gorate.Limiter)
		limiter.SetLimit(gorate.Limit(perSensorRate))
		limiter.SetBurst(bucketCapacity)
		return true
	})

	PerSensorRate.WithLabelValues(l.workloadName).Set(perSensorRate)
	PerSensorBucketCapacity.WithLabelValues(l.workloadName).Set(float64(bucketCapacity))

	log.Debugf("Rebalanced %s rate limiters: %d sensors, %.2f req/s each, burst %d",
		l.workloadName, numSensors, perSensorRate, bucketCapacity)
}

// OnSensorDisconnect removes a sensor from rate limiting and rebalances remaining limiters.
// This should be called when a sensor connection is terminated.
func (l *Limiter) OnSensorDisconnect(sensorID string) {
	if l.globalRate <= 0 {
		// Rate limiting disabled
		return
	}

	// Check if this sensor was tracked
	if _, ok := l.buckets.Load(sensorID); !ok {
		return
	}

	l.buckets.Delete(sensorID)

	numSensors := l.countActiveSensors()
	log.Infof("Sensor %s disconnected from %s rate limiting (remaining sensors: %d)",
		sensorID, l.workloadName, numSensors)

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
