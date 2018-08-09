package datastore

import (
	"fmt"
	"sort"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/search"
	"github.com/stackrox/rox/central/alert/store"
	"github.com/stackrox/rox/generated/api/v1"
	searchCommon "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (ds *datastoreImpl) SearchListAlerts(request *v1.ParsedSearchRequest) ([]*v1.ListAlert, error) {
	return ds.searcher.SearchListAlerts(request)
}

func (ds *datastoreImpl) ListAlert(id string) (*v1.ListAlert, bool, error) {
	return ds.storage.ListAlert(id)
}

func (ds *datastoreImpl) ListAlerts(request *v1.ListAlertsRequest) ([]*v1.ListAlert, error) {
	var alerts []*v1.ListAlert
	var err error
	if request.GetQuery() == "" {
		alerts, err = ds.SearchListAlerts(&v1.ParsedSearchRequest{})
		if err != nil {
			return nil, err
		}
	} else {
		parser := &searchCommon.QueryParser{}
		parsedQuery, err := parser.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, err
		}
		alerts, err = ds.SearchListAlerts(parsedQuery)
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

// CountAlerts returns the number of alerts that are active
func (ds *datastoreImpl) CountAlerts() (int, error) {
	alerts, err := ds.searcher.SearchListAlerts(
		searchCommon.NewQueryBuilder().
			AddBools(searchCommon.Stale, false).
			ToParsedSearchRequest())
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

func (ds *datastoreImpl) MarkAlertStale(id string) error {
	alert, exists, err := ds.GetAlert(id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Alert with id '%s' does not exist", id)
	}
	alert.Stale = true
	alert.MarkedStale = types.TimestampNow()
	return ds.UpdateAlert(alert)
}
