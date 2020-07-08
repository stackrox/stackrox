package fetcher

import (
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/pkg/concurrency"
)

// K8sIstioCVEManager is the interface for k8s and istio CVEs
type K8sIstioCVEManager interface {
	Start()
	Update(zipPath string, forceUpdate bool)
	HandleClusterConnection()
	GetNVDCVE(id string) *schema.NVDCVEFeedJSON10DefCVEItem
}

// k8sIstioCVEManagerImpl manages the state of k8s and istio CVEs
type k8sIstioCVEManagerImpl struct {
	k8sCVEMgr   *k8sCVEManager
	istioCVEMgr *istioCVEManager

	updateSignal concurrency.Signal
	mgrMode      mode
}

// Newk8sIstioCVEManagerImpl returns new instance of k8sIstioCVEManagerImpl
func Newk8sIstioCVEManagerImpl(clusterDataStore clusterDataStore.DataStore, cveDataStore cveDataStore.DataStore, cveMatcher *cveMatcher.CVEMatcher) (K8sIstioCVEManager, error) {
	m := &k8sIstioCVEManagerImpl{
		k8sCVEMgr: &k8sCVEManager{
			nvdCVEs:          make(map[string]*schema.NVDCVEFeedJSON10DefCVEItem),
			clusterDataStore: clusterDataStore,
			cveDataStore:     cveDataStore,
			cveMatcher:       cveMatcher,
		},
		istioCVEMgr: &istioCVEManager{
			nvdCVEs:          make(map[string]*schema.NVDCVEFeedJSON10DefCVEItem),
			clusterDataStore: clusterDataStore,
			cveDataStore:     cveDataStore,
			cveMatcher:       cveMatcher,
		},
		updateSignal: concurrency.NewSignal(),
	}

	m.initialize()
	return m, nil
}
