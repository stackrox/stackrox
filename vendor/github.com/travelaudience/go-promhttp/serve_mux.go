// Package promhttp provides handy wrappers around http package objects that allow monitoring using prometheus.
package promhttp

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	pph "github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	subsystemHTTPOutgoing = "http_outgoing"
	subsystemHTTPIncoming = "http_incoming"
)

type serveMux interface {
	Handle(pattern string, handler http.Handler)
	ServeHTTP(response http.ResponseWriter, request *http.Request)
}

// ServeMux is a wrapper for an http.ServeMux that provides Prometheus
// instrumentation for various metrics such as total requests.
type ServeMux struct {
	ServeMux serveMux

	metrics   map[string]*incomingInstrumentation
	Namespace string
}

// Handle wraps a normal http.Handle function and instruments it
// for Prometheus.
func (sm *ServeMux) Handle(path string, h http.Handler) {
	if sm.metrics == nil {
		sm.metrics = make(map[string]*incomingInstrumentation)
	}

	constLabels := map[string]string{
		"path": path,
	}
	commonLabels := []string{"code", "method"}
	ins := &incomingInstrumentation{
		duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   sm.Namespace,
				Subsystem:   subsystemHTTPIncoming,
				Name:        "request_duration_histogram_seconds",
				Help:        "Request time duration.",
				Buckets:     prometheus.DefBuckets,
				ConstLabels: constLabels,
			},
			commonLabels,
		),
		requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   sm.Namespace,
				Subsystem:   subsystemHTTPIncoming,
				Name:        "requests_total",
				Help:        "Total number of requests received.",
				ConstLabels: constLabels,
			},
			commonLabels,
		),
		requestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   sm.Namespace,
				Subsystem:   subsystemHTTPIncoming,
				Name:        "request_size_histogram_bytes",
				Help:        "Request size in bytes.",
				Buckets:     []float64{100, 1000, 2000, 5000, 10000},
				ConstLabels: constLabels,
			},
			commonLabels,
		),
		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{

				Namespace:   sm.Namespace,
				Subsystem:   subsystemHTTPIncoming,
				Name:        "response_size_histogram_bytes",
				Help:        "Response size in bytes.",
				Buckets:     []float64{100, 1000, 2000, 5000, 10000},
				ConstLabels: constLabels,
			},
			commonLabels,
		),
		inflight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   sm.Namespace,
				Subsystem:   subsystemHTTPIncoming,
				Name:        "in_flight_requests",
				Help:        "Number of http requests which are currently running.",
				ConstLabels: constLabels,
			},
		),
	}
	chain := pph.InstrumentHandlerDuration(
		ins.duration,
		pph.InstrumentHandlerCounter(
			ins.requests,
			pph.InstrumentHandlerRequestSize(
				ins.requestSize,
				pph.InstrumentHandlerResponseSize(
					ins.responseSize,
					pph.InstrumentHandlerInFlight(
						ins.inflight,
						h,
					),
				),
			)),
	)

	sm.ServeMux.Handle(path, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		chain.ServeHTTP(rw, r)
	}))
	sm.metrics[path] = ins
}

// HandleFunc registers the handler function for the given pattern. It uses instrumented Handler implementation.
func (sm *ServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	sm.Handle(pattern, http.HandlerFunc(handler))
}

// ServeHTTP implements http Handler interface.
func (sm *ServeMux) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	sm.ServeMux.ServeHTTP(rw, r)
}

// Describe implements prometheus.Collector interface.
func (sm *ServeMux) Describe(in chan<- *prometheus.Desc) {
	for _, col := range sm.metrics {
		col.describe(in)
	}
}

// Collect implements prometheus.Collector interface.
func (sm *ServeMux) Collect(in chan<- prometheus.Metric) {
	for _, col := range sm.metrics {
		col.collect(in)
	}
}

type incomingInstrumentation struct {
	duration     *prometheus.HistogramVec
	requests     *prometheus.CounterVec
	requestSize  *prometheus.HistogramVec
	responseSize *prometheus.HistogramVec
	inflight     prometheus.Gauge
}

func (i *incomingInstrumentation) describe(in chan<- *prometheus.Desc) {
	i.duration.Describe(in)
	i.requests.Describe(in)
	i.requestSize.Describe(in)
	i.responseSize.Describe(in)
	i.inflight.Describe(in)
}

func (i *incomingInstrumentation) collect(in chan<- prometheus.Metric) {
	i.duration.Collect(in)
	i.requests.Collect(in)
	i.requestSize.Collect(in)
	i.responseSize.Collect(in)
	i.inflight.Collect(in)
}
