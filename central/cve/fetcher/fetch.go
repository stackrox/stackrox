package fetcher

import (
	"fmt"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	manager K8sIstioCveManager
	once    sync.Once
)

// K8sIstioCveManager is the interface for k8s and istio CVEs
type K8sIstioCveManager interface {
	Fetch()
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

// Fetch fetches new CVEs and reconciles them
func (m *k8sIstioCveManager) Fetch() {
	for {
		if err := m.reconcileCVEs(converter.K8s); err != nil {
			log.Errorf("fetcher failed k8s CVEs with error %v", err)
		}
		if err := m.reconcileCVEs(converter.Istio); err != nil {
			log.Errorf("fetcher failed istio CVEs with error %v", err)
		}

		time.Sleep(fetchDelay)
	}
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

func (m *k8sIstioCveManager) reconcileCVEs(ct converter.CveType) error {
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

	log.Infof("successfully reconciled %d new %s CVEs", len(newCVEs), cveTypeToString[ct])
	return nil
}
