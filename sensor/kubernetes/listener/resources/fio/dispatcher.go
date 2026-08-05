package fio

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/policyreport"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var log = logging.LoggerForModule()

// DetectFunc is a callback for security event detection. Type alias so callers
// can pass the same function used for PolicyReport detection.
type DetectFunc = func(event *policyreport.SecurityEvent)

// Dispatcher processes FileIntegrityNodeStatus CRD events.
type Dispatcher struct {
	clusterID  string
	detectFunc DetectFunc
}

// NewDispatcher creates a FileIntegrityNodeStatus dispatcher.
func NewDispatcher(clusterID string, detectFunc DetectFunc) *Dispatcher {
	return &Dispatcher{
		clusterID:  clusterID,
		detectFunc: detectFunc,
	}
}

// ProcessEvent implements resources.Dispatcher.
func (d *Dispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	if !features.PolicyReports.Enabled() {
		return nil
	}

	u, ok := toUnstructured(obj)
	if !ok {
		reportsProcessed.WithLabelValues("error").Inc()
		return nil
	}

	if action == central.ResourceAction_REMOVE_RESOURCE {
		reportsProcessed.WithLabelValues("removed").Inc()
		return nil
	}

	events, err := policyreport.CanonicalizeFIO(d.clusterID, u)
	if err != nil {
		reportsProcessed.WithLabelValues("error").Inc()
		log.Warnf("Failed to canonicalize FileIntegrityNodeStatus %s/%s: %v", u.GetNamespace(), u.GetName(), err)
		return nil
	}

	reportsProcessed.WithLabelValues("ok").Inc()
	eventsCanonicalizedTotal.Add(float64(len(events)))

	if len(events) > 0 {
		log.Debugf("Canonicalized %d SecurityEvents from FileIntegrityNodeStatus %s/%s",
			len(events), u.GetNamespace(), u.GetName())
	}

	if d.detectFunc != nil {
		for i := range events {
			d.detectFunc(&events[i])
		}
	}

	return nil
}

func toUnstructured(obj interface{}) (*unstructured.Unstructured, bool) {
	u, ok := obj.(*unstructured.Unstructured)
	if ok {
		return u, true
	}
	log.Errorf("FIO dispatcher received non-Unstructured object: %T", obj)
	return nil, false
}

var (
	reportsProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "rox",
		Subsystem: "sensor",
		Name:      "fio_reports_processed_total",
		Help:      "Total FileIntegrityNodeStatus objects processed, by outcome.",
	}, []string{"outcome"})

	eventsCanonicalizedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "rox",
		Subsystem: "sensor",
		Name:      "fio_events_canonicalized_total",
		Help:      "Total SecurityEvents produced from FileIntegrityNodeStatus canonicalization.",
	})
)

func init() {
	prometheus.MustRegister(reportsProcessed, eventsCanonicalizedTotal)
}
