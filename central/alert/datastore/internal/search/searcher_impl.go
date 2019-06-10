package search

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
)

var (
	log = logging.LoggerForModule()

	defaultSortOption = &v1.SortOption{
		Field:    search.ViolationTime.String(),
		Reversed: true,
	}

	alertSearchHelper = sac.ForResource(resources.Alert).MustCreateSearchHelper(mappings.OptionsMap, sac.ClusterIDAndNamespaceFields)
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage           store.Store
	indexer           index.Indexer
	formattedSearcher search.Searcher
}

// SearchAlerts retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchAlerts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	alerts, results, err := ds.searchListAlerts(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(alerts))
	for i, alert := range alerts {
		protoResults = append(protoResults, convertAlert(alert, results[i]))
	}
	return protoResults, nil
}

// SearchRawAlerts retrieves Alerts from the indexer and storage
func (ds *searcherImpl) SearchListAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, error) {
	alerts, _, err := ds.searchListAlerts(ctx, q)
	return alerts, err
}

// SearchRawAlerts retrieves Alerts from the indexer and storage
func (ds *searcherImpl) SearchRawAlerts(ctx context.Context, q *v1.Query) ([]*storage.Alert, error) {
	alerts, err := ds.searchAlerts(ctx, q)
	return alerts, err
}

func (ds *searcherImpl) searchListAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, []search.Result, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	alerts, missingIndices, err := ds.storage.GetListAlerts(search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return alerts, results, nil
}

func (ds *searcherImpl) searchAlerts(ctx context.Context, q *v1.Query) ([]*storage.Alert, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	alerts, _, err := ds.storage.GetAlerts(search.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// Search takes a SearchRequest and finds any matches
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.formattedSearcher.Search(ctx, q)
}

// ConvertAlert returns proto search result from an alert object and the internal search result
func convertAlert(alert *storage.ListAlert, result search.Result) *v1.SearchResult {
	deployment := alert.GetDeployment()
	return &v1.SearchResult{
		Category:       v1.SearchCategory_ALERTS,
		Id:             alert.GetId(),
		Name:           alert.GetPolicy().GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
		Location:       fmt.Sprintf("/%s/%s/%s", deployment.GetClusterName(), deployment.GetNamespace(), deployment.GetName()),
	}
}

// Helper functions which format our searching.
///////////////////////////////////////////////

func formatSearcher(unsafeSearcher search.UnsafeSearcher) search.Searcher {
	filteredSearcher := alertSearchHelper.FilteredSearcher(unsafeSearcher) // Make the UnsafeSearcher safe.

	paginatedSearcher := paginated.Paginated(filteredSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	withDefaultViolationState := withDefaultActiveViolations(defaultSortedSearcher)
	return withDefaultViolationState
}

// If no active violation field is set, add one by default.
func withDefaultActiveViolations(searcher search.Searcher) search.Searcher {
	return &defaultViolationStateSearcher{
		searcher: searcher,
	}
}

type defaultViolationStateSearcher struct {
	searcher search.Searcher
}

func (ds *defaultViolationStateSearcher) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	var querySpecifiesStateField bool
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if matchFieldQuery.MatchFieldQuery.GetField() == search.ViolationState.String() {
			querySpecifiesStateField = true
		}
	})

	// By default, set stale to false.
	if !querySpecifiesStateField {
		cq := search.ConjunctionQuery(q, search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery())
		cq.Pagination = q.GetPagination()
		q = cq
	}

	return ds.searcher.Search(ctx, q)
}
