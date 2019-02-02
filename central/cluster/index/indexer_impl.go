package index

import (
	"time"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/cluster/index/mappings"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

type indexerImpl struct {
	index bleve.Index
}

type clusterWrapper struct {
	*storage.Cluster `json:"cluster"`
	Type             string `json:"type"`
}

// AddCluster adds the cluster to the index
func (b *indexerImpl) AddCluster(cluster *storage.Cluster) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Add, "Cluster")
	return b.index.Index(cluster.GetId(), &clusterWrapper{Type: v1.SearchCategory_CLUSTERS.String(), Cluster: cluster})
}

// AddClusters adds a slice of clusters to the index
func (b *indexerImpl) AddClusters(clusters []*storage.Cluster) error {
	for _, c := range clusters {
		if err := b.AddCluster(c); err != nil {
			return err
		}
	}
	return nil
}

// DeleteCluster deletes the cluster from the index
func (b *indexerImpl) DeleteCluster(id string) error {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Remove, "Cluster")
	return b.index.Delete(id)
}

// Search takes a Query and finds any matches
func (b *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "Cluster")
	return blevesearch.RunSearchRequest(v1.SearchCategory_CLUSTERS, q, b.index, mappings.OptionsMap)
}
