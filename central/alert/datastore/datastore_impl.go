package datastore

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/central/alert/index"
	"bitbucket.org/stack-rox/apollo/central/alert/search"
	"bitbucket.org/stack-rox/apollo/central/alert/store"
	"bitbucket.org/stack-rox/apollo/central/search/options"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	searchCommon "bitbucket.org/stack-rox/apollo/pkg/search"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// SearchAlerts returns search results for the given request.
func (ds *datastoreImpl) SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchAlerts(request)
}

// SearchRawAlerts returns search results for the given request in the form of a slice of alerts.
func (ds *datastoreImpl) SearchRawAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, error) {
	return ds.searcher.SearchRawAlerts(request)
}

// GetAlert returns an alert by id.
func (ds *datastoreImpl) GetAlert(id string) (*v1.Alert, bool, error) {
	return ds.storage.GetAlert(id)
}

// GetAlerts fetches the data from the database or searches for it based on the passed filters
func (ds *datastoreImpl) GetAlerts(request *v1.ListAlertsRequest) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	var err error
	if request.GetQuery() == "" {
		alerts, err = ds.SearchRawAlerts(&v1.ParsedSearchRequest{})
		if err != nil {
			return nil, err
		}
	} else {
		parser := &searchCommon.QueryParser{
			OptionsMap: options.AllOptionsMaps,
		}
		parsedQuery, err := parser.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, err
		}
		alerts, err = ds.SearchRawAlerts(parsedQuery)
		if err != nil {
			return nil, err
		}
	}

	// Sort by descending timestamp.
	sort.SliceStable(alerts, func(i, j int) bool {
		if sI, sJ := alerts[i].GetTime().GetSeconds(), alerts[j].GetTime().GetSeconds(); sI != sJ {
			return sI > sJ
		}
		return alerts[i].GetTime().GetNanos() > alerts[j].GetTime().GetNanos()
	})
	return alerts, nil
}

// CountAlerts returns the number of alerts that are active
func (ds *datastoreImpl) CountAlerts() (int, error) {
	qb := searchCommon.NewQueryBuilder().AddBool(searchCommon.Stale, false)
	// Do not call GetAlerts because they returns full alert objects which are expensive
	alerts, err := ds.GetAlerts(&v1.ListAlertsRequest{
		Query: qb.Query(),
	})
	return len(alerts), err
}

// AddAlert inserts an alert into storage and into the indexer
func (ds *datastoreImpl) AddAlert(alert *v1.Alert) error {
	if err := ds.storage.AddAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddAlert(alert)
}

// UpdateAlert updates an alert in storage and in the indexer
func (ds *datastoreImpl) UpdateAlert(alert *v1.Alert) error {
	if err := ds.storage.UpdateAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddAlert(alert)
}

// RemoveAlert removes an alert from the storage and the indexer
func (ds *datastoreImpl) RemoveAlert(id string) error {
	if err := ds.storage.RemoveAlert(id); err != nil {
		return err
	}
	return ds.indexer.DeleteAlert(id)
}
