package datastore

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// AlertDataStore is an intermediary to AlertStorage.
type AlertDataStore interface {
	db.AlertStorage

	SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, error)
}

// NewAlertDataStore provides a new instance of AlertDataStore
func NewAlertDataStore(storage db.AlertStorage, indexer search.AlertIndex) (AlertDataStore, error) {
	ds := &alertDataStoreImpl{
		AlertStorage: storage,
		indexer:      indexer,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

// alertDataStoreImpl provides an intermediary implementation layer for AlertStorage.
type alertDataStoreImpl struct {
	// This is an embedded type so we don't have to override all functions. Indexing is a subset of Storage
	db.AlertStorage

	indexer search.AlertIndex
}

func (ds *alertDataStoreImpl) buildIndex() error {
	// Alert Index
	alerts, err := ds.AlertStorage.GetAlerts(&v1.ListAlertsRequest{})
	if err != nil {
		return err
	}
	for _, a := range alerts {
		if err := ds.indexer.AddAlert(a); err != nil {
			logger.Errorf("Error inserting alert %s (%s) into index: %s", a.GetId(), a.GetPolicy().GetName(), err)
		}
	}
	return nil
}

// SearchAlerts retrieves SearchResults from the indexer and storage
func (ds *alertDataStoreImpl) SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	alerts, results, err := ds.searchAlerts(request)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(alerts))
	for i, alert := range alerts {
		protoResults = append(protoResults, search.ConvertAlert(alert, results[i]))
	}
	return protoResults, nil
}

// SearchRawAlerts retrieves Alerts from the indexer and storage
func (ds *alertDataStoreImpl) SearchRawAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, error) {
	alerts, _, err := ds.searchAlerts(request)
	return alerts, err
}

// AddAlert inserts an alert into storage and into the indexer
func (ds *alertDataStoreImpl) AddAlert(alert *v1.Alert) error {
	if err := ds.AlertStorage.AddAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddAlert(alert)
}

// UpdateAlert updates an alert in storage and in the indexer
func (ds *alertDataStoreImpl) UpdateAlert(alert *v1.Alert) error {
	if err := ds.AlertStorage.UpdateAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddAlert(alert)
}

// CountAlerts returns the number of alerts that are active
func (ds *alertDataStoreImpl) CountAlerts() (int, error) {
	qb := search.NewQueryBuilder().AddBool(search.Stale, false)
	// Do not call GetAlerts because they returns full alert objects which are expensive
	alerts, err := ds.GetAlerts(&v1.ListAlertsRequest{
		Query: qb.Query(),
	})
	return len(alerts), err
}

// RemoveAlert removes an alert from the storage and the indexer
func (ds *alertDataStoreImpl) RemoveAlert(id string) error {
	if err := ds.AlertStorage.RemoveAlert(id); err != nil {
		return err
	}
	return ds.indexer.DeleteAlert(id)
}

// GetAlerts fetches the data from the database or searches for it based on the passed filters
func (ds *alertDataStoreImpl) GetAlerts(request *v1.ListAlertsRequest) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	var err error
	if request.GetQuery() == "" {
		alerts, err = ds.SearchRawAlerts(&v1.ParsedSearchRequest{})
		if err != nil {
			return nil, err
		}
	} else {
		parsedQuery, err := search.ParseRawQuery(request.GetQuery())
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

func (ds *alertDataStoreImpl) searchAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, []search.Result, error) {
	results, err := ds.indexer.SearchAlerts(request)
	if err != nil {
		return nil, nil, err
	}
	var alerts []*v1.Alert
	var newResults []search.Result
	for _, result := range results {
		alert, exists, err := ds.GetAlert(result.ID)
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
