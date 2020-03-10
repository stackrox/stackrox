package fetcher

import (
	"context"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	"github.com/stackrox/rox/central/cve/converter"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

type istioCVEManager struct {
	nvdCVEs      map[string]*schema.NVDCVEFeedJSON10DefCVEItem
	embeddedCVEs []*storage.EmbeddedVulnerability

	clusterDataStore clusterDataStore.DataStore
	cveDataStore     cveDataStore.DataStore
	cveMatcher       *cveMatcher.CVEMatcher

	mutex sync.Mutex
}

func (m *istioCVEManager) initialize() {
	//Load the istio CVEs in mem
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

func (m *istioCVEManager) getCVEs(ctx context.Context, q *v1.Query) ([]*storage.EmbeddedVulnerability, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.filterCVEs(ctx, q)
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

func (m *istioCVEManager) filterCVEs(ctx context.Context, query *v1.Query) ([]*storage.EmbeddedVulnerability, error) {
	clusterQuery, _ := search.FilterQueryWithMap(query, clusterMappings.OptionsMap)
	clusters, err := m.clusterDataStore.SearchRawClusters(ctx, clusterQuery)
	if err != nil {
		return nil, err
	}

	return filterCVEs(ctx, query, clusters, m.embeddedCVEs, m.nvdCVEs, m.cveMatcher.IsClusterAffectedByIstioCVE)
}

func (m *istioCVEManager) updateCVEs(newCVEs []*schema.NVDCVEFeedJSON10DefCVEItem) error {
	cves, err := converter.NvdCVEsToEmbeddedCVEs(newCVEs, converter.Istio)
	if err != nil {
		return err
	}

	if !features.Dackbox.Enabled() {
		m.setCVEs(cves, newCVEs)
		return nil
	}

	m.setCVEs([]*storage.EmbeddedVulnerability{}, newCVEs)
	return m.updateCVEsInDB(cves)
}

func (m *istioCVEManager) updateCVEsInDB(embeddedCVEs []*storage.EmbeddedVulnerability) error {
	cves := converter.EmbeddedCVEsToProtoCVEs(embeddedCVEs...)
	ret := make([]converter.ClusterCVEParts, 0, len(cves))
	for _, cve := range cves {
		clusters, err := m.cveMatcher.GetAffectedClusters(m.nvdCVEs[cve.GetId()])
		if err != nil {
			return err
		}

		if len(clusters) == 0 {
			continue
		}
		ret = append(ret, converter.NewClusterCVEParts(cve, clusters, m.nvdCVEs[cve.GetId()]))
	}
	return m.cveDataStore.UpsertClusterCVEs(cveElevatedCtx, ret...)
}
