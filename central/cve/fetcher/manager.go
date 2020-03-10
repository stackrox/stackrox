package fetcher

import (
	"context"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// K8sIstioCVEManager is the interface for k8s and istio CVEs
type K8sIstioCVEManager interface {
	Fetch(forceUpdate bool)
	Update(zipPath string, forceUpdate bool)

	GetNVDCVE(id string) *schema.NVDCVEFeedJSON10DefCVEItem
	GetK8sCVEs(ctx context.Context, query *v1.Query) ([]*storage.EmbeddedVulnerability, error)
	GetIstioCVEs(ctx context.Context, query *v1.Query) ([]*storage.EmbeddedVulnerability, error)
}

// k8sIstioCVEManagerImpl manages the state of k8s and istio CVEs
type k8sIstioCVEManagerImpl struct {
	k8sCVEMgr   *k8sCVEManager
	istioCVEMgr *istioCVEManager

	mgrMode mode
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
	}

	m.initialize()
	return m, nil
}
