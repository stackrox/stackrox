package fetcher

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	manager K8sIstioCveManager
	once    sync.Once
)

type mode int

const (
	online = iota
	offline
	unknown
	k8sIstioCveZipName = "k8s-istio.zip"
)

// K8sIstioCveManager is the interface for k8s and istio CVEs
type K8sIstioCveManager interface {
	Fetch()
	Update(zipPath string)
	GetK8sCves() []*nvd.CVEEntry
	GetIstioCves() []*nvd.CVEEntry
	GetK8sAndIstioCves() []*nvd.CVEEntry
	GetK8sEmbeddedVulnerabilities() []*storage.EmbeddedVulnerability
	GetIstioEmbeddedVulnerabilities() []*storage.EmbeddedVulnerability
}

// k8sIstioCveManager manages the state of k8s and istio CVEs
type k8sIstioCveManager struct {
	k8sCveMgr   k8sCveManager
	istioCveMgr istioCveManager
	mutex       sync.Mutex
	mgrMode     mode
}

type k8sCveManager struct {
	k8sCVEs                    []*nvd.CVEEntry
	k8sEmbeddedVulnerabilities []*storage.EmbeddedVulnerability
}

type istioCveManager struct {
	istioCVEs                    []*nvd.CVEEntry
	istioEmbeddedVulnerabilities []*storage.EmbeddedVulnerability
}

// SingletonManager returns a singleton instance of k8sCveManager
func SingletonManager() K8sIstioCveManager {
	once.Do(func() {
		m := &k8sIstioCveManager{}
		utils.Must(m.initialize())
		manager = m
	})
	return manager
}

// Init copies build time CVEs to persistent volume
func (m *k8sIstioCveManager) initialize() error {
	offlineModeSetting := env.OfflineModeEnv.Setting()

	if offlineModeSetting == "true" {
		m.mgrMode = offline
	} else {
		m.mgrMode = online
	}

	if err := copyCVEsFromPreloadedToPersistentDirIfAbsent(converter.K8s); err != nil {
		return errors.Wrapf(err, "could not copy preloaded k8s CVE files to persistent volume: %q", path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir))
	}
	log.Infof("successfully copied preloaded k8s CVE files to persistent volume: %q", path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir))

	if err := copyCVEsFromPreloadedToPersistentDirIfAbsent(converter.Istio); err != nil {
		return errors.Wrapf(err, "could not copy preloaded istio CVE files to persistent volume: %q", path.Join(persistentCVEsPath, commonCveDir, istioCVEsDir))
	}
	log.Infof("successfully copied preloaded CVE istio files to persistent volume: %q", path.Join(persistentCVEsPath, commonCveDir, istioCVEsDir))

	//Load the k8s CVEs in mem
	newK8sCVEs, err := getLocalCVEs(persistentK8sCVEsFilePath)
	if err != nil {
		return err
	}
	if err := m.updateCves(newK8sCVEs, converter.K8s); err != nil {
		return err
	}
	log.Infof("successfully loaded %d k8s CVEs", len(m.GetK8sCves()))

	//Load the istio CVEs in mem
	newIstioCVEs, err := getLocalCVEs(persistentIstioCVEsFilePath)
	if err != nil {
		return err
	}
	if err := m.updateCves(newIstioCVEs, converter.Istio); err != nil {
		return err
	}
	log.Infof("successfully loaded %d istio CVEs", len(m.GetIstioCves()))

	return nil
}

// Fetch (works only in online mode) fetches new CVEs and reconciles them
func (m *k8sIstioCveManager) Fetch() {
	if m.mgrMode != online {
		log.Error("can't fetch in non-online mode")
		return
	}

	for {
		m.reconcileAllCVEsInOnlineMode()
		time.Sleep(fetchDelay)
	}
}

// Update (works only in offline mode) updates new CVEs and reconciles them based on data from scanner bundle
func (m *k8sIstioCveManager) Update(zipPath string) {
	if m.mgrMode != offline {
		log.Error("can't fetch in non-offline mode")
		return
	}

	m.reconcileAllCVEsInOfflineMode(zipPath)
}

// GetK8sCves returns current k8s CVEs loaded in memory
func (m *k8sIstioCveManager) GetK8sCves() []*nvd.CVEEntry {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.k8sCveMgr.k8sCVEs
}

// GetIstioCves returns current istio CVEs loaded in memory
func (m *k8sIstioCveManager) GetIstioCves() []*nvd.CVEEntry {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.istioCveMgr.istioCVEs
}

// GetK8sAndIstioCves returns current istio CVEs loaded in memory
func (m *k8sIstioCveManager) GetK8sAndIstioCves() []*nvd.CVEEntry {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	ret := make([]*nvd.CVEEntry, 0, len(m.k8sCveMgr.k8sCVEs)+len(m.istioCveMgr.istioCVEs))
	ret = append(ret, m.k8sCveMgr.k8sCVEs...)
	ret = append(ret, m.istioCveMgr.istioCVEs...)
	return ret
}

// GetK8sEmbeddedVulnerabilities returns the current k8s Embedded Vulns loaded in memory
func (m *k8sIstioCveManager) GetK8sEmbeddedVulnerabilities() []*storage.EmbeddedVulnerability {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.k8sCveMgr.k8sEmbeddedVulnerabilities
}

// GetIstioEmbeddedVulnerabilities returns the current istio Embedded Vulns loaded in memory
func (m *k8sIstioCveManager) GetIstioEmbeddedVulnerabilities() []*storage.EmbeddedVulnerability {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.istioCveMgr.istioEmbeddedVulnerabilities
}

func (m *k8sIstioCveManager) reconcileAllCVEsInOnlineMode() {
	if err := m.reconcileOnlineModeCVEs(converter.K8s); err != nil {
		log.Errorf("reconcile failed for k8s CVEs with error %v", err)
	}
	if err := m.reconcileOnlineModeCVEs(converter.Istio); err != nil {
		log.Errorf("reconcile failed for istio CVEs with error %v", err)
	}
}

func (m *k8sIstioCveManager) reconcileAllCVEsInOfflineMode(zipPath string) {
	if err := m.reconcileOfflineModeCVEs(converter.K8s, zipPath); err != nil {
		log.Errorf("reconcile failed for k8s CVEs with error %v", err)
	}
	if err := m.reconcileOfflineModeCVEs(converter.Istio, zipPath); err != nil {
		log.Errorf("reconcile failed for istio CVEs with error %v", err)
	}
}

func (m *k8sIstioCveManager) updateCves(newCVEs []*nvd.CVEEntry, ct converter.CveType) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	newEmbeddedVulns, err := converter.NvdCVEsToEmbeddedVulnerabilities(newCVEs, ct)
	if err != nil {
		return err
	}
	if ct == converter.K8s {
		m.k8sCveMgr.k8sCVEs = newCVEs
		m.k8sCveMgr.k8sEmbeddedVulnerabilities = newEmbeddedVulns
	} else if ct == converter.Istio {
		m.istioCveMgr.istioCVEs = newCVEs
		m.istioCveMgr.istioEmbeddedVulnerabilities = newEmbeddedVulns
	} else {
		return fmt.Errorf("unknown CVE type: %d", ct)
	}
	return nil
}

// reconcileOnlineModeCVEs fetches new CVEs from definitions.stackrox.io and reconciles them
func (m *k8sIstioCveManager) reconcileOnlineModeCVEs(ct converter.CveType) error {
	paths, err := getPaths(ct)
	if err != nil {
		return err
	}

	urls, err := getUrls(ct)
	if err != nil {
		return err
	}

	localCveChecksum, err := getLocalCVEChecksum(paths.persistentCveChecksumFile)
	if err != nil {
		return nil
	}

	remoteCveChecksum, err := fetchRemote(urls.cveChecksumURL)
	if err != nil {
		return err
	}

	// If CVEs have been loaded before and checksums are same, no need to update CVEs
	if localCveChecksum == remoteCveChecksum {
		log.Infof("local and remote CVE checksums are same, skipping download of new %s CVEs", cveTypeToString[ct])
		return nil
	}

	data, err := fetchRemote(urls.cveURL)
	if err != nil {
		return err
	}

	if err := overwriteCVEs(paths.persistentCveFile, paths.persistentCveChecksumFile, remoteCveChecksum, data); err != nil {
		return err
	}

	newCVEs, err := getLocalCVEs(paths.persistentCveFile)
	if err != nil {
		return err
	}

	if err := m.updateCves(newCVEs, ct); err != nil {
		return err
	}

	log.Infof("%s CVEs have been updated, %d new CVEs found", cveTypeToString[ct], len(newCVEs))
	return nil
}

// reconcileOfflineModeCVEs reads the scanner bundle zip and updates the CVEs
func (m *k8sIstioCveManager) reconcileOfflineModeCVEs(ct converter.CveType, zipPath string) error {
	paths, err := getPaths(ct)
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

	if ct == converter.K8s {
		bundledCVEFile = filepath.Join(bundlePath, nvd.Feeds[nvd.Kubernetes].CVEFilename)
		bundledCVEChecksumFile = filepath.Join(bundlePath, nvd.Feeds[nvd.Kubernetes].ChecksumFilename)
	} else if ct == converter.Istio {
		bundledCVEFile = filepath.Join(bundlePath, nvd.Feeds[nvd.Istio].CVEFilename)
		bundledCVEChecksumFile = filepath.Join(bundlePath, nvd.Feeds[nvd.Istio].ChecksumFilename)
	} else {
		return fmt.Errorf("unknown CVE type: %d", ct)
	}

	oldCveChecksum, err := getLocalCVEChecksum(paths.persistentCveChecksumFile)
	if err != nil {
		return nil
	}

	newCveChecksum, err := getLocalCVEChecksum(bundledCVEChecksumFile)
	if err != nil {
		return err
	}

	// If CVEs have been loaded before and checksums are same, no need to update CVEs
	if oldCveChecksum == newCveChecksum {
		log.Infof("local and bundled CVE checksums are same, skipping reconciliation of of new %s CVEs", cveTypeToString[ct])
		return nil
	}

	data, err := ioutil.ReadFile(bundledCVEFile)
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

	if err := m.updateCves(newCVEs, ct); err != nil {
		return err
	}

	log.Infof("%s CVEs have been updated, %d new CVEs found", cveTypeToString[ct], len(newCVEs))
	return nil
}

func extractK8sIstioCVEsInScannerBundleZip(zipPath string) (string, error) {
	tmpPath, err := ioutil.TempDir("", "")
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
