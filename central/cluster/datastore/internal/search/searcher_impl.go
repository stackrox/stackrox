package search

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/cluster/index/mappings"
	"github.com/stackrox/rox/central/cluster/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()

	clusterSearchHelper = sac.ForResource(resources.Cluster).MustCreateSearchHelper(mappings.OptionsMap)
)

type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

func (ds *searcherImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	clusters, results, err := ds.searchClusters(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(clusters))
	for i, alert := range clusters {
		protoResults = append(protoResults, convertCluster(alert, results[i]))
	}
	return protoResults, nil
}

func (ds *searcherImpl) SearchClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error) {
	clusters, _, err := ds.searchClusters(ctx, q)
	return clusters, err
}

func (ds *searcherImpl) searchClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, []search.Result, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	clusters, missingIndices, err := ds.storage.GetSelectedClusters(search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return clusters, results, nil
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return clusterSearchHelper.Apply(ds.indexer.Search)(ctx, q)
}

func convertCluster(cluster *storage.Cluster, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_CLUSTERS,
		Id:             cluster.GetId(),
		Name:           cluster.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
		Location:       fmt.Sprintf("/%s", cluster.GetName()),
	}
}
