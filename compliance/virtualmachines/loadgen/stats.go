package main

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
)

type statsCollector struct {
	total   atomic.Uint64
	success atomic.Uint64
	failure atomic.Uint64
	bytes   atomic.Uint64

	mu        sync.Mutex
	histogram *hdrhistogram.Histogram
}

func newStatsCollector() *statsCollector {
	return &statsCollector{
		histogram: hdrhistogram.New(1, int64((5*time.Minute)/time.Microsecond), 3),
	}
}

func (s *statsCollector) recordSuccess(latency time.Duration, bytes int) {
	s.total.Add(1)
	s.success.Add(1)
	if bytes > 0 {
		s.bytes.Add(uint64(bytes))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.histogram.RecordValue(latency.Microseconds())
}

func (s *statsCollector) recordFailure() {
	s.total.Add(1)
	s.failure.Add(1)
}

type statsSnapshot struct {
	Total      uint64
	Success    uint64
	Failure    uint64
	Bytes      uint64
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
	Elapsed    time.Duration
	Throughput float64
}

func (s *statsCollector) snapshot(elapsed time.Duration) statsSnapshot {
	total := s.total.Load()
	throughput := 0.0
	if elapsed > 0 {
		throughput = float64(total) / elapsed.Seconds()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	p50 := durationFromHist(s.histogram, 50)
	p95 := durationFromHist(s.histogram, 95)
	p99 := durationFromHist(s.histogram, 99)

	return statsSnapshot{
		Total:      total,
		Success:    s.success.Load(),
		Failure:    s.failure.Load(),
		Bytes:      s.bytes.Load(),
		P50:        p50,
		P95:        p95,
		P99:        p99,
		Elapsed:    elapsed,
		Throughput: throughput,
	}
}

func durationFromHist(h *hdrhistogram.Histogram, percentile float64) time.Duration {
	if h.TotalCount() == 0 {
		return 0
	}
	return time.Microsecond * time.Duration(h.ValueAtQuantile(percentile))
}

func logSnapshot(prefix string, snap statsSnapshot) {
	mbSent := float64(snap.Bytes) / (1024.0 * 1024.0)
	successRate := 0.0
	if snap.Elapsed > 0 {
		successRate = float64(snap.Success) / snap.Elapsed.Seconds()
	}
	log.Printf("[%s] sent=%d success=%d failure=%d throughput=%.2f req/s success_rate=%.2f req/s data=%.2f MiB p50=%s p95=%s p99=%s",
		prefix, snap.Total, snap.Success, snap.Failure, snap.Throughput, successRate, mbSent, snap.P50, snap.P95, snap.P99)
}
