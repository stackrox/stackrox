package policyreport

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

// Dispatcher processes PolicyReport CRD events, canonicalizes them into
// SecurityEvents, and records dry-run metrics. It does not forward events
// to Central — that wiring comes in a later PR once the protobuf contract
// and detector path exist.
type Dispatcher struct {
	clusterID string
}

// NewDispatcher creates a PolicyReport dispatcher.
func NewDispatcher(clusterID string) *Dispatcher {
	return &Dispatcher{clusterID: clusterID}
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
		log.Debugf("PolicyReport %s/%s removed", u.GetNamespace(), u.GetName())
		return nil
	}

	events, err := policyreport.CanonicalizeKyvernoV1Alpha2(d.clusterID, u)
	if err != nil {
		reportsProcessed.WithLabelValues("error").Inc()
		log.Warnf("Failed to canonicalize PolicyReport %s/%s: %v", u.GetNamespace(), u.GetName(), err)
		return nil
	}

	reportsProcessed.WithLabelValues("ok").Inc()
	eventsCanonicalizedTotal.Add(float64(len(events)))

	if len(events) > 0 {
		log.Debugf("Canonicalized %d actionable SecurityEvents from PolicyReport %s/%s",
			len(events), u.GetNamespace(), u.GetName())
	}

	// No forwarding to Central yet — dry-run metrics only.
	return nil
}

func toUnstructured(obj interface{}) (*unstructured.Unstructured, bool) {
	u, ok := obj.(*unstructured.Unstructured)
	if ok {
		return u, true
	}
	log.Errorf("PolicyReport dispatcher received non-Unstructured object: %T", obj)
	return nil, false
}

var (
	reportsProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "rox",
		Subsystem: "sensor",
		Name:      "policyreport_reports_processed_total",
		Help:      "Total PolicyReport objects processed by the canonicalizer, by outcome (ok, error, removed).",
	}, []string{"outcome"})

	eventsCanonicalizedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "rox",
		Subsystem: "sensor",
		Name:      "policyreport_events_canonicalized_total",
		Help:      "Total actionable SecurityEvents produced from PolicyReport canonicalization.",
	})
)

func init() {
	prometheus.MustRegister(reportsProcessed, eventsCanonicalizedTotal)
}
