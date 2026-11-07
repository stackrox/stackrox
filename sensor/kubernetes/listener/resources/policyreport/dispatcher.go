package policyreport

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/policyreport"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/references"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var log = logging.LoggerForModule()

// Dispatcher processes PolicyReport CRD events, canonicalizes them into
// SecurityEvents, resolves Pod subjects to ACS deployments, and records
// dry-run metrics. It does not forward events to Central — that wiring
// comes in a later PR once the protobuf contract and detector path exist.
type Dispatcher struct {
	clusterID       string
	hierarchy       references.ParentHierarchy
	deploymentStore store.DeploymentStore
}

// NewDispatcher creates a PolicyReport dispatcher.
func NewDispatcher(clusterID string, hierarchy references.ParentHierarchy, deploymentStore store.DeploymentStore) *Dispatcher {
	return &Dispatcher{
		clusterID:       clusterID,
		hierarchy:       hierarchy,
		deploymentStore: deploymentStore,
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

	for i := range events {
		d.resolveSubject(&events[i])
	}

	if len(events) > 0 {
		log.Debugf("Canonicalized %d actionable SecurityEvents from PolicyReport %s/%s",
			len(events), u.GetNamespace(), u.GetName())
	}

	// No forwarding to Central yet — dry-run metrics only.
	return nil
}

// resolveSubject attempts to resolve a SecurityEvent's Pod subject to an ACS
// Deployment using the parent ownership hierarchy. This mirrors the resolution
// logic in deployments.go's processWithType.
func (d *Dispatcher) resolveSubject(event *policyreport.SecurityEvent) {
	if event.Subject.Kind != "Pod" || event.Subject.UID == "" {
		resolutionTotal.WithLabelValues("unresolved").Inc()
		return
	}

	ownerIDs := d.hierarchy.TopLevelParents(event.Subject.UID)
	// TopLevelParents returns the child itself when unknown, so this is
	// effectively unreachable. Kept for defensive parity with deployments.go.
	if ownerIDs.Cardinality() == 0 {
		resolutionTotal.WithLabelValues("unresolved").Inc()
		log.Debugf("No owning deployment found for Pod %s/%s (UID %s)",
			event.Subject.Namespace, event.Subject.Name, event.Subject.UID)
		return
	}

	if ownerIDs.Cardinality() > 1 {
		resolutionTotal.WithLabelValues("error").Inc()
		log.Warnf("Multiple owning deployments for Pod %s/%s (UID %s), skipping resolution",
			event.Subject.Namespace, event.Subject.Name, event.Subject.UID)
		return
	}

	ownerID := ownerIDs.GetArbitraryElem()
	deployment := d.deploymentStore.Get(ownerID)
	if deployment == nil {
		resolutionTotal.WithLabelValues("unresolved").Inc()
		log.Debugf("Owning deployment %s not in store for Pod %s/%s",
			ownerID, event.Subject.Namespace, event.Subject.Name)
		return
	}

	event.ResolvedEntity = policyreport.ResolvedEntity{
		Type: policyreport.EntityTypeDeployment,
		ID:   deployment.GetId(),
	}
	resolutionTotal.WithLabelValues("resolved").Inc()
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

	resolutionTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "rox",
		Subsystem: "sensor",
		Name:      "policyreport_resolution_total",
		Help:      "Total Pod-to-Deployment resolution attempts, by outcome (resolved, unresolved, error).",
	}, []string{"outcome"})
)

func init() {
	prometheus.MustRegister(reportsProcessed, eventsCanonicalizedTotal, resolutionTotal)
}
