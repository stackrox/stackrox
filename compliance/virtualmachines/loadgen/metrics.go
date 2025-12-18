package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metricsRegistry struct {
	requests *prometheus.CounterVec
	bytes    prometheus.Counter
	latency  prometheus.Histogram
}

func newMetricsRegistry() *metricsRegistry {
	m := &metricsRegistry{
		requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vsock_loadgen_requests_total",
				Help: "Total vsock load generator requests by result.",
			},
			[]string{"result"},
		),
		bytes: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "vsock_loadgen_bytes_total",
				Help: "Total bytes sent to the relay.",
			},
		),
		latency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "vsock_loadgen_request_latency_seconds",
				Help:    "Request latency in seconds.",
				Buckets: prometheus.ExponentialBuckets(0.01, 1.4, 15),
			},
		),
	}
	prometheus.MustRegister(m.requests, m.bytes, m.latency)
	return m
}

func (m *metricsRegistry) observeSuccess(latency time.Duration, bytes int) {
	m.requests.WithLabelValues("success").Inc()
	m.bytes.Add(float64(bytes))
	m.latency.Observe(latency.Seconds())
}

func (m *metricsRegistry) observeFailure(reason string) {
	m.requests.WithLabelValues(reason).Inc()
}

func serveMetrics(ctx context.Context, port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Infof("metrics server listening on :%d", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Errorf("metrics server error: %v", err)
	}
}
