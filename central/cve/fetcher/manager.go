package fetcher

import (
	"context"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	"github.com/stackrox/rox/central/cve/converter"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgScanners "github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/clairify"
	"github.com/stackrox/rox/pkg/scanners/types"
)

// OrchestratorIstioCVEManager is the interface for orchestrator (k8s or openshift) and istio CVEs
type OrchestratorIstioCVEManager interface {
	Start()
	Update(zipPath string, forceUpdate bool)
	HandleClusterConnection()
	GetAffectedClusters(ctx context.Context, cveID string, ct converter.CVEType, cveMatcher *cveMatcher.CVEMatcher) ([]*storage.Cluster, error)
	UpsertOrchestratorIntegration(integration *storage.OrchestratorIntegration) error
	RemoveIntegration(integrationID string)
}

// orchestratorIstioCVEManagerImpl manages the state of orchestrator and istio CVEs
type orchestratorIstioCVEManagerImpl struct {
	orchestratorCVEMgr *orchestratorCVEManager
	istioCVEMgr        *istioCVEManager

	updateSignal    concurrency.Signal
	mgrMode         mode
	lastUpdatedTime time.Time
}

// NewOrchestratorIstioCVEManagerImpl returns new instance of orchestratorIstioCVEManagerImpl
func NewOrchestratorIstioCVEManagerImpl(clusterDataStore clusterDataStore.DataStore, cveDataStore cveDataStore.DataStore, clusterCVEDataStore clusterCVEEdgeDataStore.DataStore, cveMatcher *cveMatcher.CVEMatcher) (OrchestratorIstioCVEManager, error) {
	m := &orchestratorIstioCVEManagerImpl{
		orchestratorCVEMgr: &orchestratorCVEManager{
			clusterDataStore:    clusterDataStore,
			cveDataStore:        cveDataStore,
			clusterCVEDataStore: clusterCVEDataStore,
			cveMatcher:          cveMatcher,
			creators:            make(map[string]pkgScanners.OrchestratorScannerCreator),
			scanners:            make(map[string]types.OrchestratorScanner),
		},
		istioCVEMgr: &istioCVEManager{
			nvdCVEs:             make(map[string]*schema.NVDCVEFeedJSON10DefCVEItem),
			clusterDataStore:    clusterDataStore,
			cveDataStore:        cveDataStore,
			clusterCVEDataStore: clusterCVEDataStore,
			cveMatcher:          cveMatcher,
		},
		updateSignal: concurrency.NewSignal(),
	}
	clairifyName, clairifyCreator := clairify.OrchestratorScannerCreator()
	m.orchestratorCVEMgr.creators[clairifyName] = clairifyCreator

	m.initialize()
	return m, nil
}
