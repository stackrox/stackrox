package lifecycle

import (
	"time"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	deploymentFlushTickerDuration = 10 * time.Second

	deploymentPendingEnrichmentMaxTTL = 15 * time.Minute
)

type pendingEnrichmentDeployment struct {
	deployment    *storage.Deployment
	enrichmentCtx enricher.EnrichmentContext
	injector      common.MessageInjector
	inserted      time.Time
}

func newDeploymentsPendingEnrichment(m *managerImpl) *deploymentsPendingEnrichment {
	dpe := &deploymentsPendingEnrichment{
		manager:                      m,
		pendingEnrichmentDeployments: make(map[string]pendingEnrichmentDeployment),
		deploymentFlushTicker:        time.NewTicker(deploymentFlushTickerDuration),
	}
	go dpe.flushPeriodically()
	return dpe
}

type deploymentsPendingEnrichment struct {
	manager *managerImpl

	pendingEnrichmentDeployments     map[string]pendingEnrichmentDeployment
	pendingEnrichmentDeploymentsLock sync.Mutex
	deploymentFlushTicker            *time.Ticker
}

func (d *deploymentsPendingEnrichment) existsWithNonNilinjectorNoLock(deploymentID string) bool {
	dpe, ok := d.pendingEnrichmentDeployments[deploymentID]
	return ok && dpe.injector != nil
}

func (d *deploymentsPendingEnrichment) add(enrichmentCtx enricher.EnrichmentContext, deployment *storage.Deployment, injector common.MessageInjector) {
	d.pendingEnrichmentDeploymentsLock.Lock()
	defer d.pendingEnrichmentDeploymentsLock.Unlock()
	if d.existsWithNonNilinjectorNoLock(deployment.GetId()) {
		return
	}
	d.pendingEnrichmentDeployments[deployment.GetId()] = pendingEnrichmentDeployment{deployment: deployment, enrichmentCtx: enrichmentCtx, injector: injector, inserted: time.Now()}
}

// remove the deployment pending enrichment if we see the same deployment again.
// However, we take special case not to remove anything pending enrichment which has a non-nil injector
// because that would clobber enforcement.
func (d *deploymentsPendingEnrichment) maybeRemove(deploymentID string) {
	d.pendingEnrichmentDeploymentsLock.Lock()
	defer d.pendingEnrichmentDeploymentsLock.Unlock()
	if d.existsWithNonNilinjectorNoLock(deploymentID) {
		return
	}
	delete(d.pendingEnrichmentDeployments, deploymentID)
}

func (d *deploymentsPendingEnrichment) flushPeriodically() {
	defer d.deploymentFlushTicker.Stop()
	for range d.deploymentFlushTicker.C {
		d.flush()
	}
}

func (d *deploymentsPendingEnrichment) flush() {
	var deploymentsPendingEnrichmentCopy map[string]pendingEnrichmentDeployment
	concurrency.WithLock(&d.pendingEnrichmentDeploymentsLock, func() {
		deploymentsPendingEnrichmentCopy = make(map[string]pendingEnrichmentDeployment, len(d.pendingEnrichmentDeployments))
		for id, dep := range d.pendingEnrichmentDeployments {
			deploymentsPendingEnrichmentCopy[id] = dep
		}
	})

	for deploymentID, dpe := range deploymentsPendingEnrichmentCopy {
		enrichmentPending, err := d.manager.processDeploymentUpdate(dpe.enrichmentCtx, dpe.deployment, dpe.injector)
		if err != nil {
			log.Errorf("Failed to process pending deployment %s/%s/%s: %v", dpe.deployment.GetClusterName(), dpe.deployment.GetNamespace(), dpe.deployment.GetName(), err)
			continue
		}
		// Yay! We processed it.
		if !enrichmentPending {
			d.remove(deploymentID)
		}
		// Don't hold on to these for too long.
		if time.Since(dpe.inserted) > deploymentPendingEnrichmentMaxTTL {
			d.remove(deploymentID)
		}
	}
}

func (d *deploymentsPendingEnrichment) remove(deploymentID string) {
	d.pendingEnrichmentDeploymentsLock.Lock()
	defer d.pendingEnrichmentDeploymentsLock.Unlock()
	delete(d.pendingEnrichmentDeployments, deploymentID)
}
