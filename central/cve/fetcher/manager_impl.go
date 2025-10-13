package fetcher

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/cve/converter/utils"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/throttle"
)

const (
	minNewScannerReconcileInterval = 10 * time.Minute
)

var (
	allAccessCtx = sac.WithAllAccess(context.Background())

	connectionDropThrottle = throttle.NewDropThrottle(10 * time.Minute)

	log = logging.LoggerForModule()
)

func (m *orchestratorIstioCVEManagerImpl) initialize() {
	m.orchestratorCVEMgr.initialize()
}

// Start begins the process to periodically scan orchestrator-level components, asynchronously.
func (m *orchestratorIstioCVEManagerImpl) Start() {
	go func() {
		ticker := time.NewTicker(env.OrchestratorVulnScanInterval.DurationSetting())
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.reconcileAllCVEs()
			case <-m.updateSignal.Done():
				m.updateSignal.Reset()
				m.reconcileAllCVEs()
			}
		}
	}()
}

func (m *orchestratorIstioCVEManagerImpl) HandleClusterConnection() {
	connectionDropThrottle.Run(func() {
		m.updateSignal.Signal()
	})
}

// GetAffectedClusters returns the affected clusters for a CVE
func (m *orchestratorIstioCVEManagerImpl) GetAffectedClusters(ctx context.Context, cveID string, ct utils.CVEType, _ *cveMatcher.CVEMatcher) ([]*storage.Cluster, error) {
	clusters, err := m.orchestratorCVEMgr.getAffectedClusters(ctx, cveID, ct)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func (m *orchestratorIstioCVEManagerImpl) reconcile() {
	m.orchestratorCVEMgr.Reconcile()
}

func (m *orchestratorIstioCVEManagerImpl) reconcileAllCVEs() {
	log.Info("Start orchestrator-level vulnerability reconciliation")
	m.reconcile()
}

// UpsertOrchestratorIntegration creates or updates an orchestrator integration.
func (m *orchestratorIstioCVEManagerImpl) UpsertOrchestratorIntegration(integration *storage.OrchestratorIntegration) error {
	err := m.orchestratorCVEMgr.UpsertOrchestratorScanner(integration)
	if err != nil {
		return err
	}

	// Trigger orchestrator scan if the first scanner joins or the last scan is more than minNewScannerReconcileInterval before.
	if time.Now().After(m.lastUpdatedTime.Add(minNewScannerReconcileInterval)) {
		m.reconcile()
	}
	return nil
}

// RemoveIntegration creates or updates a node integration.
func (m *orchestratorIstioCVEManagerImpl) RemoveIntegration(integrationID string) {
	m.orchestratorCVEMgr.RemoveIntegration(integrationID)
}
