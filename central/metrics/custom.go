package metrics

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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

// RegisterCustomAggregatedMetric registers user-defined aggregated metrics
// according to the system private configuration.
func RegisterCustomAggregatedMetric(name string, description string, period time.Duration, labels []string, registryName string, exposure Exposure) error {
	// Unregister changed or disabled metric.
	if old, loaded := customAggregatedMetrics.LoadAndDelete(name); loaded && (period == 0 || exposure == NONE) {
		oldRecord := old.(*metricRecord)
		oldExposure := oldRecord.exposure
		if oldExposure == INTERNAL || oldExposure == BOTH {
			prometheus.DefaultRegisterer.Unregister(oldRecord.gauge)
		}
		if oldExposure == EXTERNAL || oldExposure == BOTH {
			GetExternalRegistry(registryName).Unregister(oldRecord.gauge)
		}
		return nil
	}

	newRecord := &metricRecord{nil, labels, registryName, exposure}

	// Register new metric, fail on an update attempt.
	if old, loaded := customAggregatedMetrics.LoadOrStore(name, newRecord); loaded {
		if !old.(*metricRecord).Equals(newRecord) {
			return fmt.Errorf("cannot update %q metric", name)
		}
	} else {
		// TODO: ensure safe concurrent access:
		newRecord.gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metrics.PrometheusNamespace,
			Subsystem: metrics.CentralSubsystem.String(),
			Name:      name,
			Help: "The total number of " + description + " aggregated by " + strings.Join(labels, ",") +
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
