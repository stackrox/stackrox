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

var (
	// Those are dynamically defined metrics, configured by users via the system
	// private configuration.
	customAggregatedMetrics sync.Map // [string]metricRecord

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
	gauge  *prometheus.GaugeVec
	labels []string
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
	metric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.CentralSubsystem.String(),
		Name:      name,
		Help: "The total number of " + description + " aggregated by " + strings.Join(labels, ",") +
			" and gathered every " + period.String(),
	}, labels)

	// Unregister disabled metric.
	if period == 0 {
		if old, loaded := customAggregatedMetrics.LoadAndDelete(name); loaded {
			if exposure == INTERNAL || exposure == BOTH {
				prometheus.DefaultRegisterer.Unregister(old.(metricRecord).gauge)
			}
			if exposure == EXTERNAL || exposure == BOTH {
				GetExternalRegistry(registryName).Unregister(old.(metricRecord).gauge)
			}
		}
		return nil
	}

	// Register new metric, alert on a labels update attempt.
	if actual, loaded := customAggregatedMetrics.LoadOrStore(name, metricRecord{metric, labels}); loaded {
		if slices.Compare(actual.(metricRecord).labels, labels) != 0 {
			return fmt.Errorf("cannot update %q metric labels", name)
		}
	} else {
		if exposure == INTERNAL || exposure == BOTH {
			if err := prometheus.DefaultRegisterer.Register(metric); err != nil {
				return err
			}
		}
		if exposure == EXTERNAL || exposure == BOTH {
			if err := GetExternalRegistry(registryName).Register(metric); err != nil {
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
