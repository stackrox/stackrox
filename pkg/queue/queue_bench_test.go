package queue

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
)

func newCounterVec(name string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "test",
		Subsystem: "queue",
		Name:      name,
	}, []string{"Operation"})
}

func newDroppedCounter(name string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "test",
		Subsystem: "queue",
		Name:      name,
	})
}

// BenchmarkPush_NotFull measures Push throughput when the queue has room.
func BenchmarkPush_NotFull(b *testing.B) {
	for _, withMetrics := range []bool{false, true} {
		name := "no_metrics"
		if withMetrics {
			name = "with_metrics"
		}
		b.Run(name, func(b *testing.B) {
			opts := []OptionFunc[int]{WithMaxSize[int](b.N + 1)}
			if withMetrics {
				opts = append(opts,
					WithCounterVec[int](newCounterVec(fmt.Sprintf("push_notfull_%s_%d", name, b.N))),
					WithDroppedMetric[int](newDroppedCounter(fmt.Sprintf("push_notfull_dropped_%s_%d", name, b.N))),
				)
			}
			q := NewQueue[int](opts...)
			b.ResetTimer()
			for i := range b.N {
				q.Push(i)
			}
		})
	}
}

// BenchmarkPush_Full measures Push throughput when the queue is at capacity
// (the drop path with logging and metrics).
func BenchmarkPush_Full(b *testing.B) {
	for _, withMetrics := range []bool{false, true} {
		name := "no_metrics"
		if withMetrics {
			name = "with_metrics"
		}
		b.Run(name, func(b *testing.B) {
			opts := []OptionFunc[int]{
				WithMaxSize[int](1),
				WithQueueName[int]("BenchQueue"),
			}
			if withMetrics {
				opts = append(opts,
					WithCounterVec[int](newCounterVec(fmt.Sprintf("push_full_%s_%d", name, b.N))),
					WithDroppedMetric[int](newDroppedCounter(fmt.Sprintf("push_full_dropped_%s_%d", name, b.N))),
				)
			}
			q := NewQueue[int](opts...)
			q.Push(0) // fill the queue
			b.ResetTimer()
			for i := range b.N {
				q.Push(i)
			}
		})
	}
}

// BenchmarkPull measures Pull throughput from a pre-filled queue.
func BenchmarkPull(b *testing.B) {
	for _, withMetrics := range []bool{false, true} {
		name := "no_metrics"
		if withMetrics {
			name = "with_metrics"
		}
		b.Run(name, func(b *testing.B) {
			opts := []OptionFunc[int]{}
			if withMetrics {
				opts = append(opts,
					WithCounterVec[int](newCounterVec(fmt.Sprintf("pull_%s_%d", name, b.N))),
				)
			}
			q := NewQueue[int](opts...)
			for i := range b.N {
				q.Push(i)
			}
			b.ResetTimer()
			for range b.N {
				q.Pull()
			}
		})
	}
}

// BenchmarkPushPull_Concurrent measures concurrent Push/Pull throughput under
// contention from multiple goroutines. This is the scenario closest to the
// production issue: many producers pushing while consumers pull.
func BenchmarkPushPull_Concurrent(b *testing.B) {
	for _, numProducers := range []int{1, 4, 16, 64} {
		for _, withMetrics := range []bool{false, true} {
			mname := "no_metrics"
			if withMetrics {
				mname = "with_metrics"
			}
			name := fmt.Sprintf("producers=%d/%s", numProducers, mname)
			b.Run(name, func(b *testing.B) {
				opts := []OptionFunc[int]{
					WithMaxSize[int](1024),
					WithQueueName[int]("BenchQueue"),
				}
				if withMetrics {
					opts = append(opts,
						WithCounterVec[int](newCounterVec(fmt.Sprintf("conc_%d_%s_%d", numProducers, mname, b.N))),
						WithDroppedMetric[int](newDroppedCounter(fmt.Sprintf("conc_drop_%d_%s_%d", numProducers, mname, b.N))),
					)
				}
				q := NewQueue[int](opts...)
				opsPerProducer := b.N / numProducers
				if opsPerProducer == 0 {
					opsPerProducer = 1
				}

				b.ResetTimer()

				var done atomic.Bool
				// Consumer goroutine draining the queue.
				go func() {
					for !done.Load() {
						q.Pull()
						runtime.Gosched()
					}
				}()

				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						q.Push(i)
						i++
					}
				})
				done.Store(true)
			})
		}
	}
}

// BenchmarkPush_Full_Concurrent measures the specific production scenario:
// many goroutines pushing to a completely full queue (all take the drop path).
// This is the exact scenario from the bug report where mutex hold time exceeded 10s.
func BenchmarkPush_Full_Concurrent(b *testing.B) {
	for _, numProducers := range []int{1, 4, 16, 64, 256} {
		for _, withMetrics := range []bool{false, true} {
			mname := "no_metrics"
			if withMetrics {
				mname = "with_metrics"
			}
			name := fmt.Sprintf("producers=%d/%s", numProducers, mname)
			b.Run(name, func(b *testing.B) {
				opts := []OptionFunc[int]{
					WithMaxSize[int](1),
					WithQueueName[int]("BenchQueue"),
				}
				if withMetrics {
					opts = append(opts,
						WithCounterVec[int](newCounterVec(fmt.Sprintf("fullconc_%d_%s_%d", numProducers, mname, b.N))),
						WithDroppedMetric[int](newDroppedCounter(fmt.Sprintf("fullconc_drop_%d_%s_%d", numProducers, mname, b.N))),
					)
				}
				q := NewQueue[int](opts...)
				q.Push(0) // fill the queue

				runtime.GOMAXPROCS(runtime.NumCPU())
				b.SetParallelism(numProducers)
				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						q.Push(i)
						i++
					}
				})
			})
		}
	}
}

// BenchmarkPush_NotFull_WithPrometheusLabels isolates the cost of
// prometheus.CounterVec.With(Labels{}).Inc() inside vs outside the lock.
// The queue code calls this on every successful Push and Pull.
func BenchmarkPush_NotFull_WithPrometheusLabels(b *testing.B) {
	cv := newCounterVec(fmt.Sprintf("labels_bench_%d", b.N))
	_ = cv.With(prometheus.Labels{"Operation": metrics.Add.String()}) // pre-warm
	b.ResetTimer()
	for range b.N {
		cv.With(prometheus.Labels{"Operation": metrics.Add.String()}).Inc()
	}
}

// BenchmarkRateLimitedLogger isolates the cost of the rate-limited logger call
// that happens on the drop path (full queue).
func BenchmarkRateLimitedLogger(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		logging.GetRateLimitedLogger().WarnL(loggingRateLimiter,
			"Queue (%s) size limit reached (%d). New items added to the queue will be dropped.",
			"BenchQueue", 40960)
	}
}
