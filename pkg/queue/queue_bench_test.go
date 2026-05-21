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

var benchSeq atomic.Int64

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, benchSeq.Add(1))
}

func newCounterVec(prefix string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "test",
		Subsystem: "queue",
		Name:      uniqueName(prefix),
	}, []string{"Operation"})
}

func newDroppedCounter(prefix string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "test",
		Subsystem: "queue",
		Name:      uniqueName(prefix),
	})
}

// BenchmarkPush_NotFull measures pure Push throughput at various starting depths.
// The queue has no max size, so it grows freely without hitting the drop path.
func BenchmarkPush_NotFull(b *testing.B) {
	for _, depth := range []int{100, 1000, 10_000, 100_000} {
		for _, withMetrics := range []bool{false, true} {
			mname := "no_metrics"
			if withMetrics {
				mname = "with_metrics"
			}
			b.Run(fmt.Sprintf("depth=%d/%s", depth, mname), func(b *testing.B) {
				opts := []OptionFunc[int]{}
				if withMetrics {
					opts = append(opts,
						WithCounterVec[int](newCounterVec("push_notfull")),
						WithDroppedMetric[int](newDroppedCounter("push_notfull_dropped")),
					)
				}
				q := NewQueue[int](opts...)
				for i := range depth {
					q.Push(i)
				}
				i := 0
				for b.Loop() {
					q.Push(i)
					i++
				}
			})
		}
	}
}

// BenchmarkPush_Full measures Push throughput when the queue is at capacity
// (the drop path with logging and metrics) at various capacities.
func BenchmarkPush_Full(b *testing.B) {
	for _, capacity := range []int{1, 100, 1000} {
		for _, withMetrics := range []bool{false, true} {
			mname := "no_metrics"
			if withMetrics {
				mname = "with_metrics"
			}
			b.Run(fmt.Sprintf("capacity=%d/%s", capacity, mname), func(b *testing.B) {
				opts := []OptionFunc[int]{
					WithMaxSize[int](capacity),
					WithQueueName[int]("BenchQueue"),
				}
				if withMetrics {
					opts = append(opts,
						WithCounterVec[int](newCounterVec("push_full")),
						WithDroppedMetric[int](newDroppedCounter("push_full_dropped")),
					)
				}
				q := NewQueue[int](opts...)
				for i := range capacity {
					q.Push(i)
				}
				i := 0
				for b.Loop() {
					q.Push(i)
					i++
				}
			})
		}
	}
}

// BenchmarkPull measures Pull throughput at various queue depths.
// Each iteration pulls one item and pushes it back to maintain constant depth.
func BenchmarkPull(b *testing.B) {
	for _, size := range []int{100, 1000, 10_000, 100_000} {
		for _, withMetrics := range []bool{false, true} {
			mname := "no_metrics"
			if withMetrics {
				mname = "with_metrics"
			}
			b.Run(fmt.Sprintf("depth=%d/%s", size, mname), func(b *testing.B) {
				opts := []OptionFunc[int]{}
				if withMetrics {
					opts = append(opts,
						WithCounterVec[int](newCounterVec("pull")),
					)
				}
				q := NewQueue[int](opts...)
				for i := range size {
					q.Push(i)
				}
				for b.Loop() {
					q.Push(q.Pull())
				}
			})
		}
	}
}

// BenchmarkPushPull_Concurrent measures concurrent Push/Pull throughput under
// contention from multiple goroutines at various queue capacities.
// This is the scenario closest to the production issue.
func BenchmarkPushPull_Concurrent(b *testing.B) {
	for _, capacity := range []int{100, 1000, 10_000} {
		for _, numProducers := range []int{1, 4, 16, 64} {
			for _, withMetrics := range []bool{false, true} {
				mname := "no_metrics"
				if withMetrics {
					mname = "with_metrics"
				}
				name := fmt.Sprintf("capacity=%d/producers=%d/%s", capacity, numProducers, mname)
				b.Run(name, func(b *testing.B) {
					opts := []OptionFunc[int]{
						WithMaxSize[int](capacity),
						WithQueueName[int]("BenchQueue"),
					}
					if withMetrics {
						opts = append(opts,
							WithCounterVec[int](newCounterVec("conc")),
							WithDroppedMetric[int](newDroppedCounter("conc_drop")),
						)
					}
					q := NewQueue[int](opts...)

					b.SetParallelism(numProducers)

					var done atomic.Bool
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
}

// BenchmarkPush_Full_Concurrent measures the specific production scenario:
// many goroutines pushing to a completely full queue (all take the drop path)
// at various capacities. This is the exact scenario from the bug report where
// mutex hold time exceeded 10s.
func BenchmarkPush_Full_Concurrent(b *testing.B) {
	for _, capacity := range []int{1, 100, 1000} {
		for _, numProducers := range []int{1, 4, 16, 64, 256} {
			for _, withMetrics := range []bool{false, true} {
				mname := "no_metrics"
				if withMetrics {
					mname = "with_metrics"
				}
				name := fmt.Sprintf("capacity=%d/producers=%d/%s", capacity, numProducers, mname)
				b.Run(name, func(b *testing.B) {
					opts := []OptionFunc[int]{
						WithMaxSize[int](capacity),
						WithQueueName[int]("BenchQueue"),
					}
					if withMetrics {
						opts = append(opts,
							WithCounterVec[int](newCounterVec("fullconc")),
							WithDroppedMetric[int](newDroppedCounter("fullconc_drop")),
						)
					}
					q := NewQueue[int](opts...)
					for i := range capacity {
						q.Push(i)
					}

					runtime.GOMAXPROCS(runtime.NumCPU())
					b.SetParallelism(numProducers)

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
}

// BenchmarkPush_NotFull_WithPrometheusLabels isolates the cost of
// prometheus.CounterVec.With(Labels{}).Inc() inside vs outside the lock.
// The queue code calls this on every successful Push and Pull.
func BenchmarkPush_NotFull_WithPrometheusLabels(b *testing.B) {
	cv := newCounterVec("labels_bench")
	_ = cv.With(prometheus.Labels{"Operation": metrics.Add.String()})
	for b.Loop() {
		cv.With(prometheus.Labels{"Operation": metrics.Add.String()}).Inc()
	}
}

// BenchmarkRateLimitedLogger isolates the cost of the rate-limited logger call
// that happens on the drop path (full queue).
func BenchmarkRateLimitedLogger(b *testing.B) {
	for b.Loop() {
		logging.GetRateLimitedLogger().WarnL(loggingRateLimiter,
			"Queue (%s) size limit reached (%d). New items added to the queue will be dropped.",
			"BenchQueue", 40960)
	}
}
