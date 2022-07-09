package fetcher

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/cve/converter/utils"
	legacyCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/throttle"
)

var (
	allAccessCtx = sac.WithAllAccess(context.Background())

	connectionDropThrottle = throttle.NewDropThrottle(10 * time.Minute)
)

const (
	minNewScannerReconcileInterval = 10 * time.Minute
)

type mode int

const (
	online = iota
	offline
	unknown
	k8sIstioCveZipName = "k8s-istio.zip"
)

// Init copies build time CVEs to persistent volume
func (m *orchestratorIstioCVEManagerImpl) initialize() {
	if env.OfflineModeEnv.BooleanSetting() {
		m.mgrMode = offline
	} else {
		m.mgrMode = online
	}

	if err := copyCVEsFromPreloadedToPersistentDirIfAbsent(utils.Istio); err != nil {
		log.Errorf("could not copy preloaded istio CVE files to persistent volume %q: %v", path.Join(persistentCVEsPath, commonCveDir, istioCVEsDir), err)
		return
	}
	log.Infof("successfully copied preloaded CVE istio files to persistent volume: %q", path.Join(persistentCVEsPath, commonCveDir, istioCVEsDir))

	m.orchestratorCVEMgr.initialize()
	m.istioCVEMgr.initialize()
}

// Fetch (works only in online mode) fetches new CVEs and reconciles them
func (m *orchestratorIstioCVEManagerImpl) Start() {
	if m.mgrMode != online {
		log.Error("can't fetch in non-online mode")
		return
	}

	ticker := time.NewTicker(fetchDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.reconcileAllCVEsInOnlineMode(true)
		case <-m.updateSignal.Done():
			m.updateSignal.Reset()
			m.reconcileAllCVEsInOnlineMode(true)
		}
	}
}

func (m *orchestratorIstioCVEManagerImpl) HandleClusterConnection() {
	connectionDropThrottle.Run(func() {
		m.updateSignal.Signal()
	})
}

// Update (works only in offline mode) updates new CVEs and reconciles them based on data from scanner bundle
func (m *orchestratorIstioCVEManagerImpl) Update(zipPath string, forceUpdate bool) {
	if m.mgrMode != offline {
		log.Error("can't fetch in non-offline mode")
		return
	}
	m.reconcileAllCVEsInOfflineMode(zipPath, forceUpdate)
}

// GetAffectedClusters returns the affected clusters for a CVE
func (m *orchestratorIstioCVEManagerImpl) GetAffectedClusters(ctx context.Context, cveID string, ct utils.CVEType, cveMatcher *cveMatcher.CVEMatcher) ([]*storage.Cluster, error) {
	if ct == utils.K8s || ct == utils.OpenShift {
		clusters, err := m.orchestratorCVEMgr.getAffectedClusters(ctx, cveID, ct)
		if err != nil {
			return nil, err
		}
		return clusters, nil
	}
	cve := m.istioCVEMgr.getNVDCVE(cveID)
	clusters, err := cveMatcher.GetAffectedClusters(ctx, cve)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

func (m *orchestratorIstioCVEManagerImpl) reconcile() {
	m.orchestratorCVEMgr.Reconcile()
}

func (m *orchestratorIstioCVEManagerImpl) reconcileAllCVEsInOnlineMode(forceUpdate bool) {
	log.Infof("Start to reconcile all CVEs online")
	m.reconcile()
	if err := m.istioCVEMgr.reconcileOnlineModeCVEs(forceUpdate); err != nil {
		log.Errorf("reconcile failed for istio CVEs with error %v", err)
	}
}

func (m *orchestratorIstioCVEManagerImpl) reconcileAllCVEsInOfflineMode(zipPath string, forceUpdate bool) {
	m.reconcile()
	if err := m.istioCVEMgr.reconcileOfflineModeCVEs(zipPath, forceUpdate); err != nil {
		log.Errorf("reconcile failed for istio CVEs with error %v", err)
	}
}

func extractK8sIstioCVEsInScannerBundleZip(zipPath string) (string, error) {
	tmpPath, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}

	if err := unzip(zipPath, tmpPath); err != nil {
		return "", err
	}

	k8sIstioZipPath := filepath.Join(tmpPath, k8sIstioCveZipName)
	if err := unzip(k8sIstioZipPath, tmpPath); err != nil {
		return "", err
	}

	return tmpPath, nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), f.Mode()); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
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
