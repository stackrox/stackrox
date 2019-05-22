package datastore

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/central/alert/convert"
	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/search"
	"github.com/stackrox/rox/central/alert/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	searchCommon "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()

	defaultSortOption = &v1.SortOption{
		Field:    searchCommon.ViolationTime.String(),
		Reversed: true,
	}
)

// datastoreImpl is a transaction script with methods that provide the domain logic for CRUD uses cases for Alert
// objects.
type datastoreImpl struct {
	storage    store.Store
	indexer    index.Indexer
	searcher   search.Searcher
	keyedMutex *concurrency.KeyedMutex
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchCommon.Result, error) {
	return ds.indexer.Search(q)
}

func (ds *datastoreImpl) SearchListAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, error) {
	return ds.searcher.SearchListAlerts(q)
}

func (ds *datastoreImpl) ListAlerts(ctx context.Context, request *v1.ListAlertsRequest) ([]*storage.ListAlert, error) {
	var q *v1.Query
	if request.GetQuery() == "" {
		q = searchCommon.EmptyQuery()
	} else {
		var err error
		q, err = searchCommon.ParseRawQuery(request.GetQuery())
		if err != nil {
			return nil, err
		}
	}
	if request.GetPagination() != nil {
		q.Pagination = request.GetPagination()
	} else {
		q.Pagination = new(v1.Pagination)
	}
	if q.Pagination.GetSortOption() == nil {
		q.Pagination.SortOption = defaultSortOption
	}

	alerts, err := ds.SearchListAlerts(ctx, q)
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// SearchAlerts returns search results for the given request.
func (ds *datastoreImpl) SearchAlerts(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchAlerts(q)
}

// SearchRawAlerts returns search results for the given request in the form of a slice of alerts.
func (ds *datastoreImpl) SearchRawAlerts(ctx context.Context, q *v1.Query) ([]*storage.Alert, error) {
	return ds.searcher.SearchRawAlerts(q)
}

// GetAlertStore returns all the alerts. Mainly used for compliance checks.
func (ds *datastoreImpl) GetAlertStore(ctx context.Context) ([]*storage.ListAlert, error) {
	return ds.ListAlerts(ctx, nil)
}

// GetAlert returns an alert by id.
func (ds *datastoreImpl) GetAlert(ctx context.Context, id string) (*storage.Alert, bool, error) {
	return ds.storage.GetAlert(id)
}

// CountAlerts returns the number of alerts that are active
func (ds *datastoreImpl) CountAlerts(ctx context.Context) (int, error) {
	alerts, err := ds.searcher.SearchListAlerts(searchCommon.NewQueryBuilder().AddStrings(searchCommon.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery())
	return len(alerts), err
}

// AddAlert inserts an alert into storage and into the indexer
func (ds *datastoreImpl) AddAlert(ctx context.Context, alert *storage.Alert) error {
	ds.keyedMutex.Lock(alert.GetId())
	defer ds.keyedMutex.Unlock(alert.GetId())
	if err := ds.storage.AddAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddListAlert(convert.AlertToListAlert(alert))
}

// UpdateAlert updates an alert in storage and in the indexer
func (ds *datastoreImpl) UpdateAlert(ctx context.Context, alert *storage.Alert) error {
	ds.keyedMutex.Lock(alert.GetId())
	defer ds.keyedMutex.Unlock(alert.GetId())
	if err := ds.storage.UpdateAlert(alert); err != nil {
		return err
	}
	return ds.indexer.AddListAlert(convert.AlertToListAlert(alert))
}

func (ds *datastoreImpl) MarkAlertStale(ctx context.Context, id string) error {
	alert, exists, err := ds.GetAlert(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("alert with id '%s' does not exist", id)
	}
	alert.State = storage.ViolationState_RESOLVED
	return ds.UpdateAlert(ctx, alert)
}
