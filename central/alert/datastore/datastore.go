package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/alert/index"
	"bitbucket.org/stack-rox/apollo/central/alert/search"
	"bitbucket.org/stack-rox/apollo/central/alert/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// DataStore is an intermediary to AlertStorage.
type DataStore interface {
	SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error)
	SearchRawAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, error)
	SearchListAlerts(request *v1.ParsedSearchRequest) ([]*v1.ListAlert, error)

	ListAlert(id string) (*v1.ListAlert, bool, error)
	ListAlerts(request *v1.ListAlertsRequest) ([]*v1.ListAlert, error)

	GetAlert(id string) (*v1.Alert, bool, error)
	CountAlerts() (int, error)
	AddAlert(alert *v1.Alert) error
	UpdateAlert(alert *v1.Alert) error
	MarkAlertStale(id string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
