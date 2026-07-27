package rate

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
)

// TestComparisonReport demonstrates the throughput difference between the
// completion-based token return approach and a simulated time-based refill.
//
// Scenario: Multiple concurrent workers process VM index reports. Each report
// takes 100-500ms to process. With a low time-based refill rate (e.g., 0.3/s),
// the old limiter starves after the initial burst because tokens only refill
// at 0.3/s regardless of processing speed. The new limiter returns tokens
// on completion, so throughput tracks actual processing capacity.
func TestComparisonReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping comparison test in short mode")
	}

	configs := []struct {
		name           string
		numWorkers     int
		bucketCapacity int
		refillRate     float64
		testDuration   time.Duration
		minProcessing  time.Duration
		maxProcessing  time.Duration
	}{
		{
			name:           "realistic_low_refill",
			numWorkers:     50,
			bucketCapacity: 200,
			refillRate:     0.3,
			testDuration:   20 * time.Second,
			minProcessing:  100 * time.Millisecond,
			maxProcessing:  500 * time.Millisecond,
		},
		{
			name:           "high_concurrency_very_low_refill",
			numWorkers:     100,
			bucketCapacity: 200,
			refillRate:     0.1,
			testDuration:   20 * time.Second,
			minProcessing:  50 * time.Millisecond,
			maxProcessing:  200 * time.Millisecond,
		},
		{
			name:           "moderate_refill",
			numWorkers:     20,
			bucketCapacity: 100,
			refillRate:     1.0,
			testDuration:   20 * time.Second,
			minProcessing:  100 * time.Millisecond,
			maxProcessing:  300 * time.Millisecond,
		},
	}

	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			oldAccepted, oldRejected := runTimeBasedTest(t, cfg.numWorkers, cfg.bucketCapacity, cfg.refillRate,
				cfg.testDuration, cfg.minProcessing, cfg.maxProcessing)
			newAccepted, newRejected := runCompletionBasedTest(t, cfg.numWorkers, cfg.bucketCapacity,
				cfg.testDuration, cfg.minProcessing, cfg.maxProcessing)

			oldRate := float64(oldAccepted) / cfg.testDuration.Seconds()
			newRate := float64(newAccepted) / cfg.testDuration.Seconds()
			var speedup float64
			if oldRate > 0 {
				speedup = newRate / oldRate
			}

			t.Logf("\n")
			t.Logf("============================================================")
			t.Logf("  RATE LIMITER COMPARISON: %s", cfg.name)
			t.Logf("============================================================")
			t.Logf("  Workers:          %d", cfg.numWorkers)
			t.Logf("  Bucket capacity:  %d tokens", cfg.bucketCapacity)
			t.Logf("  Refill rate:      %.1f tokens/sec (time-based only)", cfg.refillRate)
			t.Logf("  Test duration:    %s", cfg.testDuration)
			t.Logf("  Processing time:  %s - %s per report", cfg.minProcessing, cfg.maxProcessing)
			t.Logf("")
			t.Logf("  ┌─────────────────────┬──────────────┬──────────────────┐")
			t.Logf("  │ Metric              │ Time-Based   │ Completion-Based │")
			t.Logf("  ├─────────────────────┼──────────────┼──────────────────┤")
			t.Logf("  │ Accepted            │ %12d │ %16d │", oldAccepted, newAccepted)
			t.Logf("  │ Rejected            │ %12d │ %16d │", oldRejected, newRejected)
			t.Logf("  │ Throughput (rps)     │ %12.1f │ %16.1f │", oldRate, newRate)
			t.Logf("  │ Speedup             │            - │ %15.1fx │", speedup)
			t.Logf("  └─────────────────────┴──────────────┴──────────────────┘")
			t.Logf("")
		})
	}
}

// simulateTimeBasedLimiter simulates the old time-based token bucket.
// Tokens refill at a fixed rate per second. Processing completion does NOT return tokens.
type simulateTimeBasedLimiter struct {
	mu             sync.Mutex
	available      float64
	capacity       int
	refillRate     float64
	lastRefillTime time.Time
}

func newTimeBasedLimiter(capacity int, refillRate float64) *simulateTimeBasedLimiter {
	return &simulateTimeBasedLimiter{
		available:      float64(capacity),
		capacity:       capacity,
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

func (l *simulateTimeBasedLimiter) tryConsume() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastRefillTime).Seconds()
	l.lastRefillTime = now
	l.available += elapsed * l.refillRate
	if l.available > float64(l.capacity) {
		l.available = float64(l.capacity)
	}

	if l.available >= 1.0 {
		l.available -= 1.0
		return true
	}
	return false
}

// runTimeBasedTest simulates the old time-based token bucket behavior.
// Multiple workers compete for tokens. Tokens refill at a fixed rate.
// Processing completion does NOT return tokens.
func runTimeBasedTest(t *testing.T, numWorkers, capacity int, refillRate float64, duration, minProc, maxProc time.Duration) (accepted, rejected int64) {
	t.Helper()

	limiter := newTimeBasedLimiter(capacity, refillRate)
	deadline := time.Now().Add(duration)

	var wg sync.WaitGroup
	var totalAccepted, totalRejected atomic.Int64

	for w := range numWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(workerID)))
			for time.Now().Before(deadline) {
				if limiter.tryConsume() {
					totalAccepted.Add(1)
					procTime := minProc + time.Duration(rng.Int63n(int64(maxProc-minProc)))
					time.Sleep(procTime)
					// Old approach: NO token return on completion
				} else {
					totalRejected.Add(1)
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(w)
	}

	wg.Wait()
	return totalAccepted.Load(), totalRejected.Load()
}

// runCompletionBasedTest uses the actual new Limiter with Return() on completion.
// Multiple workers compete for tokens. Tokens are returned when processing finishes.
func runCompletionBasedTest(t *testing.T, numWorkers, capacity int, duration, minProc, maxProc time.Duration) (accepted, rejected int64) {
	t.Helper()

	limiter, err := NewLimiter("vm_index_reports", 0.3, capacity).ForAllWorkloads()
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	deadline := time.Now().Add(duration)
	clientID := "sensor-1"

	// Prime the client so per-client capacity is calculated
	limiter.TryConsume(clientID, &central.MsgFromSensor{})
	limiter.Return(clientID, &central.MsgFromSensor{})

	var wg sync.WaitGroup
	var totalAccepted, totalRejected atomic.Int64

	for w := range numWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(workerID)))
			for time.Now().Before(deadline) {
				if allowed, _ := limiter.TryConsume(clientID, &central.MsgFromSensor{}); allowed {
					totalAccepted.Add(1)
					procTime := minProc + time.Duration(rng.Int63n(int64(maxProc-minProc)))
					time.Sleep(procTime)
					// New approach: return token when processing completes
					limiter.Return(clientID, &central.MsgFromSensor{})
				} else {
					totalRejected.Add(1)
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(w)
	}

	wg.Wait()
	return totalAccepted.Load(), totalRejected.Load()
}

// TestComparisonTimeSeries shows how throughput evolves over time.
// This demonstrates the "cliff" effect: time-based hits a wall after initial burst,
// while completion-based sustains throughput.
func TestComparisonTimeSeries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping time series test in short mode")
	}

	numWorkers := 50
	capacity := 100
	refillRate := 0.5
	totalDuration := 30 * time.Second
	bucketInterval := 2 * time.Second
	minProc := 100 * time.Millisecond
	maxProc := 400 * time.Millisecond

	t.Logf("Time series comparison (bucket=%s):", bucketInterval)
	t.Logf("  Workers=%d, Capacity=%d, RefillRate=%.1f/s, Processing=%s-%s",
		numWorkers, capacity, refillRate, minProc, maxProc)
	t.Logf("")

	oldBuckets := runTimeBasedTimeSeries(t, numWorkers, capacity, refillRate, totalDuration, bucketInterval, minProc, maxProc)
	newBuckets := runCompletionBasedTimeSeries(t, numWorkers, capacity, totalDuration, bucketInterval, minProc, maxProc)

	t.Logf("  ┌──────────┬──────────────────────┬──────────────────────────┐")
	t.Logf("  │ Time (s) │ Time-Based (accepted) │ Completion-Based (acc.)  │")
	t.Logf("  ├──────────┼──────────────────────┼──────────────────────────┤")
	maxBuckets := len(oldBuckets)
	if len(newBuckets) > maxBuckets {
		maxBuckets = len(newBuckets)
	}
	for i := range maxBuckets {
		oldVal := int64(0)
		newVal := int64(0)
		if i < len(oldBuckets) {
			oldVal = oldBuckets[i]
		}
		if i < len(newBuckets) {
			newVal = newBuckets[i]
		}
		timeLabel := fmt.Sprintf("%d-%d", i*int(bucketInterval.Seconds()), (i+1)*int(bucketInterval.Seconds()))
		t.Logf("  │ %8s │ %20d │ %24d │", timeLabel, oldVal, newVal)
	}
	t.Logf("  └──────────┴──────────────────────┴──────────────────────────┘")
}

func runTimeBasedTimeSeries(t *testing.T, numWorkers, capacity int, refillRate float64, totalDuration, bucketInterval, minProc, maxProc time.Duration) []int64 {
	t.Helper()

	limiter := newTimeBasedLimiter(capacity, refillRate)
	startTime := time.Now()
	deadline := startTime.Add(totalDuration)
	numBuckets := int(totalDuration / bucketInterval)
	buckets := make([]atomic.Int64, numBuckets)

	var wg sync.WaitGroup
	for w := range numWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(workerID)))
			for time.Now().Before(deadline) {
				if limiter.tryConsume() {
					elapsed := time.Since(startTime)
					bucketIdx := int(elapsed / bucketInterval)
					if bucketIdx >= numBuckets {
						bucketIdx = numBuckets - 1
					}
					buckets[bucketIdx].Add(1)
					procTime := minProc + time.Duration(rng.Int63n(int64(maxProc-minProc)))
					time.Sleep(procTime)
				} else {
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(w)
	}

	wg.Wait()
	result := make([]int64, numBuckets)
	for i := range numBuckets {
		result[i] = buckets[i].Load()
	}
	return result
}

func runCompletionBasedTimeSeries(t *testing.T, numWorkers, capacity int, totalDuration, bucketInterval, minProc, maxProc time.Duration) []int64 {
	t.Helper()

	limiter, err := NewLimiter("vm_index_reports", 0.3, capacity).ForAllWorkloads()
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	clientID := "sensor-1"
	limiter.TryConsume(clientID, &central.MsgFromSensor{})
	limiter.Return(clientID, &central.MsgFromSensor{})

	startTime := time.Now()
	deadline := startTime.Add(totalDuration)
	numBuckets := int(totalDuration / bucketInterval)
	buckets := make([]atomic.Int64, numBuckets)

	var wg sync.WaitGroup
	for w := range numWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(workerID)))
			for time.Now().Before(deadline) {
				if allowed, _ := limiter.TryConsume(clientID, &central.MsgFromSensor{}); allowed {
					elapsed := time.Since(startTime)
					bucketIdx := int(elapsed / bucketInterval)
					if bucketIdx >= numBuckets {
						bucketIdx = numBuckets - 1
					}
					buckets[bucketIdx].Add(1)
					procTime := minProc + time.Duration(rng.Int63n(int64(maxProc-minProc)))
					time.Sleep(procTime)
					limiter.Return(clientID, &central.MsgFromSensor{})
				} else {
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(w)
	}

	wg.Wait()
	result := make([]int64, numBuckets)
	for i := range numBuckets {
		result[i] = buckets[i].Load()
	}
	return result
}
