package fetcher

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	clusterCVEEdgeDataStore "github.com/stackrox/stackrox/central/clustercveedge/datastore"
	"github.com/stackrox/stackrox/central/cve/converter"
	cveDataStore "github.com/stackrox/stackrox/central/cve/datastore"
	cveMatcher "github.com/stackrox/stackrox/central/cve/matcher"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/httputil"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/urlfmt"
)

type istioCVEManager struct {
	nvdCVEs      map[string]*schema.NVDCVEFeedJSON10DefCVEItem
	embeddedCVEs []*storage.EmbeddedVulnerability

	clusterDataStore    clusterDataStore.DataStore
	cveDataStore        cveDataStore.DataStore
	clusterCVEDataStore clusterCVEEdgeDataStore.DataStore
	cveMatcher          *cveMatcher.CVEMatcher

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
	cves, err := converter.NvdCVEsToEmbeddedCVEs(newCVEs, converter.Istio)
	if err != nil {
		return err
	}

	m.setCVEs([]*storage.EmbeddedVulnerability{}, newCVEs)
	return m.updateCVEsInDB(cves)
}

func (m *istioCVEManager) updateCVEsInDB(embeddedCVEs []*storage.EmbeddedVulnerability) error {
	cves := converter.EmbeddedCVEsToProtoCVEs("", embeddedCVEs...)
	newCVEs := make([]converter.ClusterCVEParts, 0, len(cves))
	for _, cve := range cves {
		var nvdCVE *schema.NVDCVEFeedJSON10DefCVEItem
		if features.PostgresDatastore.Enabled() {
			nvdCVE = m.getNVDCVE(cve.GetCve())
		} else {
			nvdCVE = m.getNVDCVE(cve.GetId())
		}
		if nvdCVE == nil {
			continue
		}

		clusters, err := m.cveMatcher.GetAffectedClusters(cveElevatedCtx, nvdCVE)
		if err != nil {
			return err
		}

		if len(clusters) == 0 {
			continue
		}

		fixVersions := strings.Join(converter.GetFixedVersions(nvdCVE), ",")
		newCVEs = append(newCVEs, converter.NewClusterCVEParts(cve, clusters, fixVersions))
	}

	if err := m.clusterCVEDataStore.Upsert(cveElevatedCtx, newCVEs...); err != nil {
		return err
	}
	return reconcileCVEsInDB(m.cveDataStore, m.clusterCVEDataStore, storage.CVE_ISTIO_CVE, newCVEs)
}

// reconcileOnlineModeCVEs fetches new CVEs from definitions.stackrox.io and reconciles them
func (m *istioCVEManager) reconcileOnlineModeCVEs(forceUpdate bool) error {
	paths, err := getPaths(converter.Istio)
	if err != nil {
		return err
	}

	urls, err := getUrls(converter.Istio)
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
	paths, err := getPaths(converter.Istio)
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
