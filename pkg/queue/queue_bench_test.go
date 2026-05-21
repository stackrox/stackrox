package queue

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
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

// BenchmarkPush_Full measures the true per-operation cost of pushing to a full
// queue (the drop path with logging and metrics), single-threaded via b.Loop().
//
// This benchmark is intentionally separate from BenchmarkPush_Full_Concurrent
// with producers=1. Despite both using a single producer, b.RunParallel
// reports ~40% lower ns/op because it distributes iterations across
// goroutine start/stop cycles, which changes the timing of the rate-limited
// logger and gives an artificially low per-op number. b.Loop() runs all
// iterations sequentially on one goroutine, providing the honest baseline.
//
// Capacity is fixed at 1. Varying it has no effect because the drop path only
// checks list.Len() >= maxSize (O(1) regardless of size) and never touches
// the queue contents.
func BenchmarkPush_Full(b *testing.B) {
	for _, withMetrics := range []bool{false, true} {
		mname := "no_metrics"
		if withMetrics {
			mname = "with_metrics"
		}
		b.Run(mname, func(b *testing.B) {
			opts := []OptionFunc[int]{
				WithMaxSize[int](1),
				WithQueueName[int]("BenchQueue"),
			}
			if withMetrics {
				opts = append(opts,
					WithCounterVec[int](newCounterVec("push_full")),
					WithDroppedMetric[int](newDroppedCounter("push_full_dropped")),
				)
			}
			q := NewQueue(opts...)
			q.Push(0)
			i := 0
			for b.Loop() {
				q.Push(i)
				i++
			}
		})
	}
}

// BenchmarkPull measures Pull throughput on a pre-filled queue.
// Each iteration pulls one item and pushes it back to maintain constant depth.
//
// Depth is fixed at 100. The underlying container/list provides O(1)
// Front/Remove/PushBack regardless of list length, so varying depth
// has no effect on per-operation cost.
func BenchmarkPull(b *testing.B) {
	for _, withMetrics := range []bool{false, true} {
		mname := "no_metrics"
		if withMetrics {
			mname = "with_metrics"
		}
		b.Run(mname, func(b *testing.B) {
			opts := []OptionFunc[int]{}
			if withMetrics {
				opts = append(opts,
					WithCounterVec[int](newCounterVec("pull")),
				)
			}
			q := NewQueue(opts...)
			for i := range 100 {
				q.Push(i)
			}
			for b.Loop() {
				q.Push(q.Pull())
			}
		})
	}
}

// BenchmarkPush_NotFull_Concurrent measures concurrent Push throughput on an
// unbounded queue (success path only). A background goroutine drains the queue
// to prevent unbounded memory growth.
func BenchmarkPush_NotFull_Concurrent(b *testing.B) {
	for _, numProducers := range []int{1, 4, 16, 64, 256} {
		for _, withMetrics := range []bool{false, true} {
			mname := "no_metrics"
			if withMetrics {
				mname = "with_metrics"
			}
			name := fmt.Sprintf("producers=%d/%s", numProducers, mname)
			b.Run(name, func(b *testing.B) {
				opts := []OptionFunc[int]{}
				if withMetrics {
					opts = append(opts,
						WithCounterVec[int](newCounterVec("conc")),
						WithDroppedMetric[int](newDroppedCounter("conc_drop")),
					)
				}
				q := NewQueue(opts...)

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

// BenchmarkPush_Full_Concurrent measures the specific production scenario:
// many goroutines pushing to a completely full queue (all take the drop path).
// This is the exact scenario from the bug report where mutex hold time exceeded 10s.
//
// Capacity is fixed at 1. Varying it has no effect because the drop path only
// checks list.Len() >= maxSize (O(1) regardless of size) and never touches
// the queue contents.
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
						WithCounterVec[int](newCounterVec("fullconc")),
						WithDroppedMetric[int](newDroppedCounter("fullconc_drop")),
					)
				}
				q := NewQueue(opts...)
				q.Push(0)

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
