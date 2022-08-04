package fetcher

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/cve/converter/utils"
	converterV2 "github.com/stackrox/rox/central/cve/converter/v2"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
)

type istioCVEManager struct {
	nvdCVEs      map[string]*schema.NVDCVEFeedJSON10DefCVEItem
	embeddedCVEs []*storage.EmbeddedVulnerability

	clusterDataStore        clusterDataStore.DataStore
	clusterCVEDataStore     clusterCVEDataStore.DataStore
	clusterCVEEdgeDataStore clusterCVEEdgeDataStore.DataStore
	legacyCVEDataStore      cveDataStore.DataStore
	cveMatcher              *cveMatcher.CVEMatcher

	mutex sync.Mutex
}

func (m *istioCVEManager) initialize() {
	// Load the istio CVEs in mem
	newIstioCVEs, err := getLocalCVEs(persistentIstioCVEsFilePath)
	if err != nil {
		log.Errorf("failed to get local istio cves: %v", err)
		return
	}
	if err := m.updateCVEs(newIstioCVEs); err != nil {
		log.Errorf("failed to update istio cves: %v", err)
		return
	}
	log.Infof("successfully fetched %d istio CVEs", len(m.nvdCVEs))
}

func (m *istioCVEManager) getNVDCVE(id string) *schema.NVDCVEFeedJSON10DefCVEItem {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.nvdCVEs[id]
}

func (m *istioCVEManager) setCVEs(cves []*storage.EmbeddedVulnerability, nvdCVEs []*schema.NVDCVEFeedJSON10DefCVEItem) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, nvdCVE := range nvdCVEs {
		m.nvdCVEs[nvdCVE.CVE.CVEDataMeta.ID] = nvdCVE
	}
	m.embeddedCVEs = cves
}

func (m *istioCVEManager) updateCVEs(newCVEs []*schema.NVDCVEFeedJSON10DefCVEItem) error {
	cves, err := utils.NvdCVEsToEmbeddedCVEs(newCVEs, utils.Istio)
	if err != nil {
		return err
	}

	m.setCVEs([]*storage.EmbeddedVulnerability{}, newCVEs)
	if features.PostgresDatastore.Enabled() {
		return m.updateCVEsInPostgres(cves)
	}
	return m.updateCVEsInDB(cves)
}

func (m *istioCVEManager) updateCVEsInPostgres(embeddedCVEs []*storage.EmbeddedVulnerability) error {
	cves := make([]*storage.ClusterCVE, 0, len(embeddedCVEs))
	for _, from := range embeddedCVEs {
		cves = append(cves, utils.EmbeddedVulnerabilityToClusterCVE(storage.CVE_ISTIO_CVE, from))
	}

	newCVEs := make([]converterV2.ClusterCVEParts, 0, len(cves))
	for _, cve := range cves {
		nvdCVE := m.getNVDCVE(cve.GetCveBaseInfo().GetCve())
		if m.getNVDCVE(cve.GetCveBaseInfo().GetCve()) == nil {
			continue
		}

		clusters, err := m.cveMatcher.GetAffectedClusters(allAccessCtx, nvdCVE)
		if err != nil {
			return err
		}

		if len(clusters) == 0 {
			continue
		}

		fixVersions := strings.Join(utils.GetFixedVersions(nvdCVE), ",")
		newCVEs = append(newCVEs, converterV2.NewClusterCVEParts(cve, clusters, fixVersions))
	}

	return m.clusterCVEDataStore.UpsertClusterCVEsInternal(allAccessCtx, storage.CVE_ISTIO_CVE, newCVEs...)
	// Reconciliation is performed in postgres store.
}

func (m *istioCVEManager) updateCVEsInDB(embeddedCVEs []*storage.EmbeddedVulnerability) error {
	cves := utils.EmbeddedCVEsToProtoCVEs("", embeddedCVEs...)
	newCVEs := make([]converter.ClusterCVEParts, 0, len(cves))
	for _, cve := range cves {
		nvdCVE := m.getNVDCVE(cve.GetId())
		if nvdCVE == nil {
			continue
		}

		clusters, err := m.cveMatcher.GetAffectedClusters(allAccessCtx, nvdCVE)
		if err != nil {
			return err
		}

		if len(clusters) == 0 {
			continue
		}

		fixVersions := strings.Join(utils.GetFixedVersions(nvdCVE), ",")
		newCVEs = append(newCVEs, converter.NewClusterCVEParts(cve, clusters, fixVersions))
	}

	if err := m.clusterCVEEdgeDataStore.Upsert(allAccessCtx, newCVEs...); err != nil {
		return err
	}
	return reconcileCVEsInDB(m.legacyCVEDataStore, m.clusterCVEEdgeDataStore, storage.CVE_ISTIO_CVE, newCVEs)
}

// reconcileOnlineModeCVEs fetches new CVEs from definitions.stackrox.io and reconciles them
func (m *istioCVEManager) reconcileOnlineModeCVEs(forceUpdate bool) error {
	paths, err := getPaths(utils.Istio)
	if err != nil {
		return err
	}

	urls, err := getUrls(utils.Istio)
	if err != nil {
		return err
	}

	localCVEChecksum, err := getLocalCVEChecksum(paths.persistentCveChecksumFile)
	if err != nil {
		return nil
	}

	endpoint, err := urlfmt.FullyQualifiedURL(urls.cveURL, url.Values{})
	if err != nil {
		return err
	}

	remoteCVEChecksumBytes, err := httputil.HTTPGet(endpoint)
	if err != nil {
		return err
	}

	remoteCVEChecksum := string(remoteCVEChecksumBytes)
	// If CVEs have been loaded before and checksums are same, no need to update CVEs
	if !forceUpdate && localCVEChecksum == remoteCVEChecksum {
		log.Info("local and remote CVE checksums are same, skipping download of new Istio CVEs")
		return nil
	}

	endpoint, err = urlfmt.FullyQualifiedURL(urls.cveURL, url.Values{})
	if err != nil {
		return err
	}

	data, err := httputil.HTTPGet(endpoint)
	if err != nil {
		return err
	}

	if err := overwriteCVEs(paths.persistentCveFile, paths.persistentCveChecksumFile, remoteCVEChecksum, string(data)); err != nil {
		return err
	}

	newCVEs, err := getLocalCVEs(paths.persistentCveFile)
	if err != nil {
		return err
	}

	if err := m.updateCVEs(newCVEs); err != nil {
		return err
	}

	if localCVEChecksum != remoteCVEChecksum {
		log.Infof("Istio CVEs have been updated, %d new CVEs found", len(newCVEs))
	}
	return nil
}

// reconcileOfflineModeCVEs reads the scanner bundle zip and updates the CVEs
func (m *istioCVEManager) reconcileOfflineModeCVEs(zipPath string, forceUpdate bool) error {
	paths, err := getPaths(utils.Istio)
	if err != nil {
		return err
	}

	bundlePath, err := extractK8sIstioCVEsInScannerBundleZip(zipPath)
	if err != nil {
		return err
	}
	defer func() {
		err := os.RemoveAll(bundlePath)
		if err != nil {
			log.Errorf("error while deleting the temp bundle dir, error: %v", err)
		}
	}()

	var bundledCVEFile, bundledCVEChecksumFile string

	bundledCVEFile = filepath.Join(bundlePath, nvd.Feeds[nvd.Istio].CVEFilename)
	bundledCVEChecksumFile = filepath.Join(bundlePath, nvd.Feeds[nvd.Istio].ChecksumFilename)

	oldCveChecksum, err := getLocalCVEChecksum(paths.persistentCveChecksumFile)
	if err != nil {
		return nil
	}

	newCveChecksum, err := getLocalCVEChecksum(bundledCVEChecksumFile)
	if err != nil {
		return err
	}

	// If CVEs have been loaded before and checksums are same, no need to update CVEs
	if !forceUpdate && oldCveChecksum == newCveChecksum {
		log.Infof("local and bundled CVE checksums are same, skipping reconciliation of of new Istio CVEs")
		return nil
	}

	data, err := os.ReadFile(bundledCVEFile)
	if err != nil {
		return err
	}

	if err := overwriteCVEs(paths.persistentCveFile, paths.persistentCveChecksumFile, newCveChecksum, string(data)); err != nil {
		return err
	}

	newCVEs, err := getLocalCVEs(paths.persistentCveFile)
	if err != nil {
		return err
	}

	if err := m.updateCVEs(newCVEs); err != nil {
		return err
	}

	if oldCveChecksum != newCveChecksum {
		log.Infof("Istio CVEs have been updated, %d new CVEs found", len(newCVEs))
	}
	return nil
}
