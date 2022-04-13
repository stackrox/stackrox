package metrics

import (
	"reflect"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/stackrox/generated/internalapi/central"
)

// GetResourceString takes in a sensor event and returns a resource string to be used in Prometheus
func GetResourceString(event *central.SensorEvent) string {
	resourceType := "none"
	if event.GetResource() != nil {
		resourceType = strings.TrimPrefix(reflect.TypeOf(event.GetResource()).Elem().Name(), "SensorEvent_")
	}
	return resourceType
}

// SetResourceProcessingDurationForEvent gets the resource timing from the Timing field on the event
func SetResourceProcessingDurationForEvent(metric *prometheus.HistogramVec, event *central.SensorEvent, typ string) {
	timing := event.GetTiming()
	if timing == nil {
		return
	}
	now := time.Now().UnixNano()
	diff := now - timing.GetNanos()
	// Potentially clock skew between Central and Sensor
	if diff < 0 {
		return
	}
	labels := prometheus.Labels{
		"Action":     event.GetAction().String(),
		"Resource":   timing.GetResource(),
		"Dispatcher": timing.GetDispatcher(),
	}
	if typ != "" {
		labels["Type"] = typ
	}
	metric.With(labels).Observe(float64(diff / int64(time.Millisecond)))
}
