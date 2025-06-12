package metrics

import (
	"slices"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

/*
Custom Prometheus metrics are user facing metrics, served on the API endpoint
or/and on the internal central metrics endpoint. The metrics are configured via
the private section of the 'config' API.

A metric is configured with a list of labels and the registry name, which is
appended to the endpoint's /metrics path to support tenant isolation by URL. For
example: /metrics/cluster1 to show only metrics with labels filtered for
cluster1. The label expressions are fully under user control.

When a user updates the configuration for a custom Prometheus metric (such as
changing its labels, exposure, or registry name), the system must ensure that
the metric is correctly registered or unregistered from the appropriate
Prometheus registries.

Metrics are immutable. To change anything for a metric: registry, exposure,
or labels, the metric has to be deleted and recreated with separate requests.
*/

var (
	// Those are dynamically defined metrics, configured by users via the system
	// private configuration.
	customAggregatedMetrics sync.Map // [string]*metricRecord
	ErrChangedExposure      = errox.InvalidArgs.New("changed exposure")

	customRegistries = map[string]*prometheus.Registry{"": prometheus.NewRegistry()}
	regMux           sync.RWMutex
)

type Exposure uint8

const (
	NONE Exposure = iota
	INTERNAL
	EXTERNAL
	BOTH
)

type metricRecord struct {
	gauge    *prometheus.GaugeVec
	labels   []string
	registry string
	exposure Exposure
}

func (mr *metricRecord) Equals(rec *metricRecord) bool {
	return mr == nil && rec == nil ||
		mr != nil && rec != nil &&
			mr.registry == rec.registry &&
			mr.exposure == rec.exposure &&
			slices.Compare(mr.labels, rec.labels) == 0
}

func CheckExposureChange(name string, registry string, exposure Exposure) error {
	if v, ok := customAggregatedMetrics.Load(name); ok {
		mr := v.(*metricRecord)
		if mr.exposure != exposure || mr.registry != registry {
			return ErrChangedExposure
		}
	}
	return nil
}

func GetExternalRegistry(name string) *prometheus.Registry {
	regMux.Lock()
	defer regMux.Unlock()
	r, ok := customRegistries[name]
	if !ok {
		r = prometheus.NewRegistry()
		customRegistries[name] = r
	}
	return r
}

func IsKnownRegistry(name string) bool {
	regMux.RLock()
	defer regMux.RUnlock()
	_, ok := customRegistries[name]
	return ok
}

func UnregisterCustomAggregatedMetric(name string) bool {
	v, ok := customAggregatedMetrics.LoadAndDelete(name)
	if !ok {
		return false
	}
	mr := v.(*metricRecord)
	e := mr.exposure
	if e == INTERNAL || e == BOTH {
		prometheus.DefaultRegisterer.Unregister(mr.gauge)
	}
	if e == EXTERNAL || e == BOTH {
		GetExternalRegistry(mr.registry).Unregister(mr.gauge)
	}
	return true
}

// RegisterCustomAggregatedMetric registers user-defined aggregated metrics
// according to the system private configuration.
func RegisterCustomAggregatedMetric(name string, category string, period time.Duration, labels []string, registryName string, exposure Exposure) error {
	newRecord := &metricRecord{nil, labels, registryName, exposure}

	if _, loaded := customAggregatedMetrics.LoadOrStore(name, newRecord); loaded {
		return nil
	}

	// TODO: ensure safe concurrent access:
	newRecord.gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      name,
		Help: "The total number of " + category + " aggregated by " + strings.Join(labels, ",") +
			" and gathered every " + period.String(),
	}, labels)

	if exposure == INTERNAL || exposure == BOTH {
		if err := prometheus.DefaultRegisterer.Register(newRecord.gauge); err != nil {
			return err
		}
	}
	if exposure == EXTERNAL || exposure == BOTH {
		if err := GetExternalRegistry(registryName).Register(newRecord.gauge); err != nil {
			return err
		}
	}

	return nil
}

// SetCustomAggregatedCount registers the metric vector with the values,
// according to the system private configuration.
func SetCustomAggregatedCount(metricName string, labels prometheus.Labels, total int) {
	if metric, ok := customAggregatedMetrics.Load(metricName); ok {
		metric.(metricRecord).gauge.With(labels).Set(float64(total))
	}
}
