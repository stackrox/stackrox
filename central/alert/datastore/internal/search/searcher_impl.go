package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/alert/convert"
	"github.com/stackrox/rox/pkg/search"
)

type AlertSearcher interface {
	Search(ctx context.Context, q *v1.Query, excludeResolved bool) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query, excludeResolved bool) (int, error)
}

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage           store.Store
	formattedSearcher AlertSearcher
}

// SearchAlerts retrieves SearchResults from the storage
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

// SearchListAlerts retrieves list alerts from the storage, passing excludeResolved = true will exclude resolved alerts unless the query has explicitly added Violation State = Resolved to the filter
func (ds *searcherImpl) SearchListAlerts(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*storage.ListAlert, error) {
	if excludeResolved {
		q = applyDefaultState(q)
	}
	alerts, err := ds.storage.GetByQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	listAlerts := make([]*storage.ListAlert, 0, len(alerts))
	for _, alert := range alerts {
		listAlerts = append(listAlerts, convert.AlertToListAlert(alert))
	}
	return listAlerts, nil

}

// SearchRawAlerts retrieves Alerts from the storage
func (ds *searcherImpl) SearchRawAlerts(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*storage.Alert, error) {
	alerts, err := ds.searchAlerts(ctx, q, excludeResolved)
	return alerts, err
}

func (ds *searcherImpl) searchListAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, []search.Result, error) {
	results, err := ds.Search(ctx, q, true)
	if err != nil {
		return nil, nil, err
	}
	alerts, missingIndices, err := ds.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}
	listAlerts := make([]*storage.ListAlert, 0, len(alerts))
	for _, alert := range alerts {
		listAlerts = append(listAlerts, convert.AlertToListAlert(alert))
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return listAlerts, results, nil
}

func (ds *searcherImpl) searchAlerts(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*storage.Alert, error) {
	if excludeResolved {
		q = applyDefaultState(q)
	}
	return ds.storage.GetByQuery(ctx, q)
}

// Search takes a SearchRequest and finds any matches
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query, excludeResolved bool) ([]search.Result, error) {
	return ds.formattedSearcher.Search(ctx, q, excludeResolved)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query, excludeResolved bool) (int, error) {
	return ds.formattedSearcher.Count(ctx, q, excludeResolved)
}

// convertAlert returns proto search result from an alert object and the internal search result
func convertAlert(alert *storage.ListAlert, result search.Result) *v1.SearchResult {
	entityInfo := alert.GetCommonEntityInfo()
	var entityName string
	switch entity := alert.GetEntity().(type) {
	case *storage.ListAlert_Resource:
		entityName = entity.Resource.GetName()
	case *storage.ListAlert_Deployment:
		entityName = entity.Deployment.GetName()
	}
	resourceTypeTitleCase := strings.Title(strings.ToLower(entityInfo.GetResourceType().String()))
	var location string
	if entityInfo.GetNamespace() != "" {
		location = fmt.Sprintf("/%s/%s/%s/%s",
			entityInfo.GetClusterName(), entityInfo.GetNamespace(), resourceTypeTitleCase, entityName)
	} else {
		location = fmt.Sprintf("/%s/%s/%s",
			entityInfo.GetClusterName(), resourceTypeTitleCase, entityName)
	}
	return &v1.SearchResult{
		Category:       v1.SearchCategory_ALERTS,
		Id:             alert.GetId(),
		Name:           alert.GetPolicy().GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
		Location:       location,
	}
}

// Helper functions which format our searching.
///////////////////////////////////////////////

func formatSearcher(searcher search.Searcher) AlertSearcher {
	withDefaultViolationState := withDefaultActiveViolations(searcher)
	return withDefaultViolationState
}

func applyDefaultState(q *v1.Query) *v1.Query {
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
		cq := search.ConjunctionQuery(q, search.NewQueryBuilder().AddExactMatches(
			search.ViolationState,
			storage.ViolationState_ACTIVE.String(),
			storage.ViolationState_ATTEMPTED.String()).ProtoQuery())
		cq.Pagination = q.GetPagination()
		return cq
	}
	return q
}

// If no active violation field is set, add one by default.
func withDefaultActiveViolations(searcher search.Searcher) AlertSearcher {
	return &defaultViolationStateSearcher{
		searcher: searcher,
	}
}

type defaultViolationStateSearcher struct {
	searcher search.Searcher
}

func (ds *defaultViolationStateSearcher) Search(ctx context.Context, q *v1.Query, excludeResolved bool) ([]search.Result, error) {
	if excludeResolved {
		q = applyDefaultState(q)
	}
	return ds.searcher.Search(ctx, q)
}

func (ds *defaultViolationStateSearcher) Count(ctx context.Context, q *v1.Query, excludeResolved bool) (int, error) {
	if excludeResolved {
		q = applyDefaultState(q)
	}
	return ds.searcher.Count(ctx, q)
}
