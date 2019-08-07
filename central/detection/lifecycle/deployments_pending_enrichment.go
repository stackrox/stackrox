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

type deploymentPendingEnrichment struct {
	deployment    *storage.Deployment
	enrichmentCtx enricher.EnrichmentContext
	injector      common.MessageInjector
	inserted      time.Time
}

func newDeploymentsPendingEnrichment(m *managerImpl) *deploymentsPendingEnrichment {
	dpe := &deploymentsPendingEnrichment{
		manager:    m,
		pendingMap: make(map[string]deploymentPendingEnrichment),
		ticker:     time.NewTicker(deploymentFlushTickerDuration),
	}
	go dpe.flushPeriodically()
	return dpe
}

type deploymentsPendingEnrichment struct {
	manager *managerImpl

	pendingMap map[string]deploymentPendingEnrichment
	lock       sync.Mutex
	ticker     *time.Ticker
}

func (d *deploymentsPendingEnrichment) add(enrichmentCtx enricher.EnrichmentContext, deployment *storage.Deployment, injector common.MessageInjector) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.pendingMap[deployment.GetId()] = deploymentPendingEnrichment{deployment: deployment, enrichmentCtx: enrichmentCtx, injector: injector, inserted: time.Now()}
}

// Remove the deployment pending enrichment if we see the same deployment again.
// Return the injector associated with that deployment, if any.
func (d *deploymentsPendingEnrichment) removeAndRetrieveInjector(deploymentID string) common.MessageInjector {
	d.lock.Lock()
	defer d.lock.Unlock()
	var injector common.MessageInjector
	if dpe, exists := d.pendingMap[deploymentID]; exists {
		injector = dpe.injector
	}
	delete(d.pendingMap, deploymentID)
	return injector
}

func (d *deploymentsPendingEnrichment) flushPeriodically() {
	defer d.ticker.Stop()
	for range d.ticker.C {
		d.flush()
	}
}

func (d *deploymentsPendingEnrichment) flush() {
	var deploymentsPendingEnrichmentCopy map[string]deploymentPendingEnrichment
	concurrency.WithLock(&d.lock, func() {
		deploymentsPendingEnrichmentCopy = make(map[string]deploymentPendingEnrichment, len(d.pendingMap))
		for id, dep := range d.pendingMap {
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
	d.lock.Lock()
	defer d.lock.Unlock()
	delete(d.pendingMap, deploymentID)
}
