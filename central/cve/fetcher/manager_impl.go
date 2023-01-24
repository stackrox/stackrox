package fetcher

import (
	"context"
	"time"

	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/cve/converter/utils"
	legacyCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
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
	log.Infof("Start orchestrator-level vulnerability reconciliation")
	m.reconcile()
}

func reconcileCVEsInDB(cveDataStore legacyCVEDataStore.DataStore, edgeDataStore clusterCVEEdgeDataStore.DataStore,
	cveType storage.CVE_CVEType, newCVEs []converter.ClusterCVEParts) error {
	query := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVEType, cveType.String()).ProtoQuery()
	cveResults, err := cveDataStore.Search(allAccessCtx, query)
	if err != nil {
		return err
	}

	edgeResults, err := edgeDataStore.Search(allAccessCtx, query)
	if err != nil {
		return err
	}

	// Identify the cves and cluster cve edges that do not affect the infra
	discardEdgeIds := pkgSearch.ResultsToIDSet(edgeResults)
	discardCVEs := pkgSearch.ResultsToIDSet(cveResults)

	for _, newCVE := range newCVEs {
		for _, edge := range newCVE.Children {
			discardEdgeIds.Remove(edge.Edge.GetId())
		}
		discardCVEs.Remove(newCVE.CVE.GetId())
	}

	if len(discardCVEs) == 0 && len(discardEdgeIds) == 0 {
		return nil
	}

	err = edgeDataStore.Delete(allAccessCtx, discardEdgeIds.AsSlice()...)
	if err != nil {
		return err
	}

	// delete all the cluster cves that do not affect the infra
	return cveDataStore.Delete(allAccessCtx, discardCVEs.AsSlice()...)
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
