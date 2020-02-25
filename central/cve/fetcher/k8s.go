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
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	vulnPredicateFactory = predicate.NewFactory("vulnerability", &storage.EmbeddedVulnerability{})
)

type k8sCVEManager struct {
	nvdCVEs      map[string]*schema.NVDCVEFeedJSON10DefCVEItem
	embeddedCVEs []*storage.EmbeddedVulnerability

	clusterDataStore clusterDataStore.DataStore
	cveDataStore     cveDataStore.DataStore
	cveMatcher       *cveMatcher.CVEMatcher

	mutex sync.Mutex
}

func (m *k8sCVEManager) initialize() error {
	//Load the k8s CVEs in mem
	newK8sCVEs, err := getLocalCVEs(persistentK8sCVEsFilePath)
	if err != nil {
		return err
	}
	if err := m.updateCVEs(newK8sCVEs); err != nil {
		return err
	}
	log.Infof("successfully fetched %d k8s CVEs", len(m.nvdCVEs))
	return nil
}

func (m *k8sCVEManager) getCVEs(ctx context.Context, q *v1.Query) ([]*storage.EmbeddedVulnerability, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.filterCVEs(ctx, q)
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

func (m *k8sCVEManager) filterCVEs(ctx context.Context, query *v1.Query) ([]*storage.EmbeddedVulnerability, error) {
	clusterQuery, _ := search.FilterQueryWithMap(query, clusterMappings.OptionsMap)
	clusters, err := m.clusterDataStore.SearchRawClusters(ctx, clusterQuery)
	if err != nil {
		return nil, err
	}

	return filterCVEs(ctx, query, clusters, m.embeddedCVEs, m.nvdCVEs, m.cveMatcher.IsClusterAffectedByK8sCVE)
}

func (m *k8sCVEManager) updateCVEs(newCVEs []*schema.NVDCVEFeedJSON10DefCVEItem) error {
	cves, err := converter.NvdCVEsToEmbeddedCVEs(newCVEs, converter.K8s)
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

func (m *k8sCVEManager) updateCVEsInDB(embeddedCVEs []*storage.EmbeddedVulnerability) error {
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
