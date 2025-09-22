package metrics

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

/*
Custom Prometheus metrics are user facing metrics, served on the API endpoint.
The metrics are configured via the private section of the 'config' API.

A metric is configured with a list of labels supported by the metric tracker.

Metrics are immutable. To change anything, the metric has to be deleted and
recreated with separate requests.
*/

//go:generate mockgen-wrapper
type CustomRegistry interface {
	prometheus.Gatherer
	http.Handler
	Lock()
	Unlock()
	RegisterMetric(metricName string, category string, period time.Duration, labels []string) error
	UnregisterMetric(metricName string) bool
	SetTotal(metricName string, labels prometheus.Labels, total int)
	Reset(metricName string)
}

type customRegistry struct {
	*prometheus.Registry
	sync.Mutex
	http.Handler
	gauges sync.Map // map[metricName string]*prometheus.GaugeVec
}

var (
	userRegistries map[string]*customRegistry = make(map[string]*customRegistry)
	registriesMux  sync.Mutex
)

// GetCustomRegistry is a CustomRegistry factory that returns the existing or
// a new registry for the user.
func GetCustomRegistry(userID string) CustomRegistry {
	registriesMux.Lock()
	defer registriesMux.Unlock()
	registry, ok := userRegistries[userID]
	if !ok {
		registry = &customRegistry{
			Registry: prometheus.NewRegistry(),
		}
		registry.Handler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		userRegistries[userID] = registry
	}
	return registry
}

var _ CustomRegistry = (*customRegistry)(nil)

// UnregisterMetric unregister the given metric by name.
func (cr *customRegistry) UnregisterMetric(metricName string) bool {
	if gauge, loaded := cr.gauges.LoadAndDelete(metricName); loaded {
		return cr.Unregister(gauge.(*prometheus.GaugeVec))
	}
	return false
}

// RegisterMetric registers a user-defined aggregated metric.
func (cr *customRegistry) RegisterMetric(metricName string, description string, period time.Duration, labels []string) error {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      metricName,
		Help: "The total number of " + description + " aggregated by " + strings.Join(labels, ",") +
			" and gathered every " + period.String(),
	}, labels)
	if _, loaded := cr.gauges.LoadOrStore(metricName, gauge); !loaded {
		return cr.Register(gauge)
	}
	return nil
}

// SetTotal sets the value to the gauge of a metric.
func (cr *customRegistry) SetTotal(metricName string, labels prometheus.Labels, total int) {
	if gauge, ok := cr.gauges.Load(metricName); ok {
		gauge.(*prometheus.GaugeVec).With(labels).Set(float64(total))
	}
}

// Reset the metric to drop potentially stale labels.
func (cr *customRegistry) Reset(metricName string) {
	if gauge, ok := cr.gauges.Load(metricName); ok {
		gauge.(*prometheus.GaugeVec).Reset()
	}
}
