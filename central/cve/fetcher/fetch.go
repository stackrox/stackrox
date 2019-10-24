package fetcher

import (
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	manager K8sCveManager
	once    sync.Once
)

// K8sCveManager is the interface for k8s CVEs
type K8sCveManager interface {
	Fetch()
	GetK8sCves() []nvd.CVEEntry
}

// K8sCveManager manages the state of k8s CVEs
type k8sCveManager struct {
	k8sCVEs []nvd.CVEEntry
	mutex   sync.Mutex
}

// SingletonManager returns a singleton instance of k8sCveManager
func SingletonManager() K8sCveManager {
	once.Do(func() {
		m := &k8sCveManager{}
		utils.Must(m.initialize())
		manager = m
	})
	return manager
}

// Init copies build time CVEs to persistent volume
func (m *k8sCveManager) initialize() error {
	if err := copyK8sCVEsFromPreloadedToPersistentDirIfAbsent(); err != nil {
		return errors.Wrapf(err, "could not copy preloaded cve files to persistent volume: %q", path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir))
	}
	log.Infof("successfully copied preloaded cve files to persistent volume: %q", path.Join(persistentCVEsPath, commonCveDir, k8sCVEsDir))

	//Also load the CVEs in mem
	newCVEs, err := getLocalCVEs(k8sCVEsPersistentFilePath)
	if err != nil {
		return err
	}
	m.updateK8sCves(newCVEs)
	log.Infof("successfully loaded %d cves", len(m.GetK8sCves()))
	return nil
}

// Fetch fetches new CVEs and reconciles them
func (m *k8sCveManager) Fetch() {
	for {
		if err := m.reconcileCVEs(); err != nil {
			log.Errorf("fetcher failed with error %v", err)
		}
		time.Sleep(fetchDelay)
	}
}

// GetK8sCves returns the current CVE loaded in memory
func (m *k8sCveManager) GetK8sCves() []nvd.CVEEntry {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.k8sCVEs
}

func (m *k8sCveManager) updateK8sCves(newCves []nvd.CVEEntry) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.k8sCVEs = newCves
}

func (m *k8sCveManager) reconcileCVEs() error {
	localK8sCveHash, err := getLocalCVEsHash(k8sCVEsHashPersistentFilePath)
	if err != nil {
		return nil
	}

	remoteK8sCveHash, err := fetchRemote(k8sCVEsHashURL)
	if err != nil {
		return err
	}

	// If cves have been loaded before and hashes are same, no need to update CVEs
	if localK8sCveHash == remoteK8sCveHash {
		log.Info("local and remote CVE hash are same, skipping download of new CVEs")
		return nil
	}

	data, err := fetchRemote(k8sCVEsURL)
	if err != nil {
		return err
	}

	if err := overwriteCVEs(k8sCVEsPersistentFilePath, k8sCVEsHashPersistentFilePath, remoteK8sCveHash, data); err != nil {
		return err
	}

	newCVEs, err := getLocalCVEs(k8sCVEsPersistentFilePath)
	if err != nil {
		return err
	}

	m.updateK8sCves(newCVEs)

	log.Infof("successfully reconciled %d new CVEs", len(newCVEs))
	return nil
}
