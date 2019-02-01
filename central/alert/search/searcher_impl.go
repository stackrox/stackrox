package search

import (
	"fmt"

	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

func (ds *searcherImpl) buildIndex() error {
	// Alert Index
	alerts, err := ds.storage.GetAlerts()
	if err != nil {
		return err
	}
	return ds.indexer.AddAlerts(alerts)
}

// SearchAlerts retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchAlerts(q *v1.Query) ([]*v1.SearchResult, error) {
	alerts, results, err := ds.searchListAlerts(q)
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
func (ds *searcherImpl) SearchListAlerts(q *v1.Query) ([]*storage.ListAlert, error) {
	alerts, _, err := ds.searchListAlerts(q)
	return alerts, err
}

// SearchRawAlerts retrieves Alerts from the indexer and storage
func (ds *searcherImpl) SearchRawAlerts(q *v1.Query) ([]*storage.Alert, error) {
	alerts, err := ds.searchAlerts(q)
	return alerts, err
}

func (ds *searcherImpl) searchListAlerts(q *v1.Query) ([]*storage.ListAlert, []search.Result, error) {
	results, err := ds.indexer.Search(q)
	if err != nil {
		return nil, nil, err
	}
	var alerts []*storage.ListAlert
	var newResults []search.Result
	for _, result := range results {
		alert, exists, err := ds.storage.ListAlert(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		alerts = append(alerts, alert)
		newResults = append(newResults, result)
	}
	return alerts, newResults, nil
}

func (ds *searcherImpl) searchAlerts(q *v1.Query) ([]*storage.Alert, error) {
	results, err := ds.indexer.Search(q)
	if err != nil {
		return nil, err
	}
	var alerts []*storage.Alert
	for _, result := range results {
		alert, exists, err := ds.storage.GetAlert(result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
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
