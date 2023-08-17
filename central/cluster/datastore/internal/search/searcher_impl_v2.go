package search

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/cluster/index"
	store "github.com/stackrox/rox/central/cluster/store/cluster"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sorted"
)

var (
	sacHelper         = sac.ForResource(resources.Cluster).MustCreatePgSearchHelper()
	defaultSortOption = &v1.QuerySortOption{
		Field:    search.Cluster.String(),
		Reversed: false,
	}
)

// NewV2 returns a new instance of Searcher for the given storage and indexer.
func NewV2(storage store.Store, indexer index.Indexer, clusterRanker *ranking.Ranker) Searcher {
	return &searcherImplV2{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcherV2(indexer, clusterRanker),
	}
}

func formatSearcherV2(searcher search.Searcher, clusterRanker *ranking.Ranker) search.Searcher {
	// scopedSearcher := postgres.WithScoping(sacHelper.FilteredSearcher(searcher))
	scopedSearcher := postgres.WithScoping(searcher)
	prioritySortedSearcher := sorted.Searcher(scopedSearcher, search.ClusterPriority, clusterRanker)
	paginatedSearcher := paginated.Paginated(prioritySortedSearcher)
	return paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
}

type searcherImplV2 struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (s *searcherImplV2) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	clusters, results, err := s.searchClusters(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(clusters))
	for i, cluster := range clusters {
		protoResults = append(protoResults, convertCluster(cluster, results[i]))
	}
	return protoResults, nil
}

func (s *searcherImplV2) SearchClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, error) {
	clusters, _, err := s.searchClusters(ctx, q)
	return clusters, err
}

func (s *searcherImplV2) searchClusters(ctx context.Context, q *v1.Query) ([]*storage.Cluster, []search.Result, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	clusters, missingIndices, err := s.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}

	results = search.RemoveMissingResults(results, missingIndices)
	return clusters, results, nil
}

func (s *searcherImplV2) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImplV2) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.searcher.Count(ctx, q)
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
