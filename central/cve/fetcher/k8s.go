package fetcher

import (
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/cve/converter"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

type k8sCVEManager struct {
	nvdCVEs      map[string]*schema.NVDCVEFeedJSON10DefCVEItem
	embeddedCVEs []*storage.EmbeddedVulnerability

	clusterDataStore clusterDataStore.DataStore
	cveDataStore     cveDataStore.DataStore
	cveMatcher       *cveMatcher.CVEMatcher

	mutex sync.Mutex
}

func (m *k8sCVEManager) initialize() {
	//Load the k8s CVEs in mem
	newK8sCVEs, err := getLocalCVEs(persistentK8sCVEsFilePath)
	if err != nil {
		log.Errorf("failed to get local k8s cves: %v", err)
		return
	}
	if err := m.updateCVEs(newK8sCVEs); err != nil {
		log.Errorf("failed to get update k8s cves: %v", err)
		return
	}
	log.Infof("successfully fetched %d k8s CVEs", len(m.nvdCVEs))
}

func (m *k8sCVEManager) getNVDCVE(id string) *schema.NVDCVEFeedJSON10DefCVEItem {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.nvdCVEs[id]
}

func (m *k8sCVEManager) setCVEs(cves []*storage.EmbeddedVulnerability, nvdCVEs []*schema.NVDCVEFeedJSON10DefCVEItem) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, nvdCVE := range nvdCVEs {
		m.nvdCVEs[nvdCVE.CVE.CVEDataMeta.ID] = nvdCVE
	}
	m.embeddedCVEs = cves
}

func (m *k8sCVEManager) updateCVEs(newCVEs []*schema.NVDCVEFeedJSON10DefCVEItem) error {
	cves, err := converter.NvdCVEsToEmbeddedCVEs(newCVEs, converter.K8s)
	if err != nil {
		return err
	}

	m.setCVEs([]*storage.EmbeddedVulnerability{}, newCVEs)
	return m.updateCVEsInDB(cves)
}

func (m *k8sCVEManager) updateCVEsInDB(embeddedCVEs []*storage.EmbeddedVulnerability) error {
	cves := converter.EmbeddedCVEsToProtoCVEs(embeddedCVEs...)
	newCVEs := make([]converter.ClusterCVEParts, 0, len(cves))
	newCVEIDs := set.NewStringSet()
	for _, cve := range cves {
		clusters, err := m.cveMatcher.GetAffectedClusters(m.getNVDCVE(cve.GetId()))
		if err != nil {
			return err
		}

		if len(clusters) == 0 {
			continue
		}
		newCVEIDs.Add(cve.GetId())
		newCVEs = append(newCVEs, converter.NewClusterCVEParts(cve, clusters, m.getNVDCVE(cve.GetId())))
	}

	if err := m.cveDataStore.UpsertClusterCVEs(cveElevatedCtx, newCVEs...); err != nil {
		return err
	}
	return reconcileCVEsInDB(m.cveDataStore, storage.CVE_K8S_CVE, newCVEIDs)
}
