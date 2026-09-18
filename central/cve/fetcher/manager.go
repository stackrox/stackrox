package fetcher

import (
	"context"
	"time"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	"github.com/stackrox/rox/central/cve/converter/utils"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/backgroundworker"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	pkgScanners "github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	"github.com/stackrox/rox/pkg/scanners/types"
)

// OrchestratorIstioCVEManager is the interface for orchestrator (k8s or openshift) and istio CVEs
type OrchestratorIstioCVEManager interface {
	Start()
	HandleClusterConnection()
	GetAffectedClusters(ctx context.Context, cveID string, ct utils.CVEType, cveMatcher *cveMatcher.CVEMatcher) ([]*storage.Cluster, error)
	UpsertOrchestratorIntegration(integration *storage.OrchestratorIntegration) error
	RemoveIntegration(integrationID string)
}

// orchestratorIstioCVEManagerImpl manages the state of orchestrator and istio CVEs
type orchestratorIstioCVEManagerImpl struct {
	orchestratorCVEMgr *orchestratorCVEManager

	worker          *backgroundworker.PeriodicWorker
	updateSignal    *backgroundworker.SignalChannel
	lastUpdatedTime time.Time
}

// NewOrchestratorIstioCVEManagerImpl returns new instance of orchestratorIstioCVEManagerImpl
func NewOrchestratorIstioCVEManagerImpl(
	clusterDataStore clusterDataStore.DataStore,
	clusterCVEDataStore clusterCVEDataStore.DataStore,
	clusterCVEEdgeDataStore clusterCVEEdgeDataStore.DataStore,
	cveMatcher *cveMatcher.CVEMatcher,
) (OrchestratorIstioCVEManager, error) {
	m := &orchestratorIstioCVEManagerImpl{
		orchestratorCVEMgr: &orchestratorCVEManager{
			clusterDataStore:        clusterDataStore,
			clusterCVEDataStore:     clusterCVEDataStore,
			clusterCVEEdgeDataStore: clusterCVEEdgeDataStore,
			cveMatcher:              cveMatcher,
			creators:                make(map[string]pkgScanners.OrchestratorScannerCreator),
			scanners:                make(map[string]types.OrchestratorScanner),
		},
		updateSignal: backgroundworker.NewSignalChannel(),
	}
	m.worker = &backgroundworker.PeriodicWorker{
		Name:         "orchestrator_cve_fetcher",
		Interval:     env.OrchestratorVulnScanInterval.DurationSetting(),
		ShortCircuit: m.updateSignal.C(),
		Run: func(_ context.Context) error {
			m.reconcileAllCVEs()
			return nil
		},
	}
	backgroundworker.Global.Register(m.worker)
	if !features.LegacyScanner.Enabled() {
		log.Info("Orchestrator scanning is disabled: no orchestrator scanners are integrated")
		return m, nil
	}

	clairifyName, clairifyCreator := clairify.OrchestratorScannerCreator()
	m.orchestratorCVEMgr.creators[clairifyName] = clairifyCreator

	m.initialize()
	return m, nil
}
