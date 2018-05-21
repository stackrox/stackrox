package datastore

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/set"
	ptypes "github.com/gogo/protobuf/types"
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
	alerts, err := ds.GetAlerts(&v1.GetAlertsRequest{})
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
	alerts, err := ds.GetAlerts(&v1.GetAlertsRequest{Stale: []bool{false}})
	return len(alerts), err
}

// RemoveAlert removes an alert from the storage and the indexer
func (ds *alertDataStoreImpl) RemoveAlert(id string) error {
	if err := ds.AlertStorage.RemoveAlert(id); err != nil {
		return err
	}
	return ds.indexer.DeleteAlert(id)
}

type severitiesWrap []v1.Severity

func (wrap severitiesWrap) asSet() map[v1.Severity]struct{} {
	output := make(map[v1.Severity]struct{})

	for _, s := range wrap {
		output[s] = struct{}{}
	}

	return output
}

// GetAlerts fetches the data from the database or searches for it based on the passed filters
func (ds *alertDataStoreImpl) GetAlerts(request *v1.GetAlertsRequest) ([]*v1.Alert, error) {
	var alerts []*v1.Alert
	var err error
	if request.GetQuery() == "" {
		alerts, err = ds.AlertStorage.GetAlerts(request)
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
	sinceTime, sinceTimeErr := ptypes.TimestampFromProto(request.GetSince())
	untilTime, untilTimeErr := ptypes.TimestampFromProto(request.GetUntil())
	sinceStaleTime, sinceStaleTimeErr := ptypes.TimestampFromProto(request.GetSinceStale())
	untilStaleTime, untilStaleTimeErr := ptypes.TimestampFromProto(request.GetUntilStale())

	severitySet := severitiesWrap(request.GetSeverity()).asSet()
	categorySet := set.NewSetFromStringSlice(request.GetCategory())
	filtered := alerts[:0]

	for _, alert := range alerts {
		if len(request.GetStale()) == 1 && alert.GetStale() != request.GetStale()[0] {
			continue
		}
		if request.GetDeploymentId() != "" && request.GetDeploymentId() != alert.GetDeployment().GetId() {
			continue
		}

		if request.GetPolicyId() != "" && request.GetPolicyId() != alert.GetPolicy().GetId() {
			continue
		}

		if _, ok := severitySet[alert.GetPolicy().GetSeverity()]; len(severitySet) > 0 && !ok {
			continue
		}
		alertCategoriesSet := set.NewSetFromStringSlice(alert.GetPolicy().GetCategories())
		if categorySet.Cardinality() != 0 && categorySet.Intersect(alertCategoriesSet).Cardinality() == 0 {
			continue
		}
		if sinceTimeErr == nil && !sinceTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.TimestampFromProto(alert.GetTime()); alertTimeErr == nil && !sinceTime.Before(alertTime) {
				continue
			}
		}
		if untilTimeErr == nil && !untilTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.TimestampFromProto(alert.GetTime()); alertTimeErr == nil && !untilTime.After(alertTime) {
				continue
			}
		}
		if sinceStaleTimeErr == nil && !sinceStaleTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.TimestampFromProto(alert.GetTime()); alertTimeErr == nil && !sinceStaleTime.Before(alertTime) {
				continue
			}
		}
		if untilStaleTimeErr == nil && !untilStaleTime.IsZero() {
			if alertTime, alertTimeErr := ptypes.TimestampFromProto(alert.GetTime()); alertTimeErr == nil && !untilStaleTime.After(alertTime) {
				continue
			}
		}
		filtered = append(filtered, alert)
	}
	// Sort by descending timestamp.
	sort.SliceStable(filtered, func(i, j int) bool {
		if sI, sJ := filtered[i].GetTime().GetSeconds(), filtered[j].GetTime().GetSeconds(); sI != sJ {
			return sI > sJ
		}
		return filtered[i].GetTime().GetNanos() > filtered[j].GetTime().GetNanos()
	})
	return filtered, nil
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
