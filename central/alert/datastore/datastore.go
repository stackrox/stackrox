package datastore

import (
	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/search"
	"github.com/stackrox/rox/central/alert/store"
	"github.com/stackrox/rox/generated/api/v1"
)

// DataStore is a transaction script with methods that provide the domain logic for CRUD uses cases for Alert objects.
//go:generate mockery -name=DataStore
type DataStore interface {
	SearchAlerts(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawAlerts(q *v1.Query) ([]*v1.Alert, error)
	SearchListAlerts(q *v1.Query) ([]*v1.ListAlert, error)

	ListAlerts(request *v1.ListAlertsRequest) ([]*v1.ListAlert, error)

	GetAlert(id string) (*v1.Alert, bool, error)
	CountAlerts() (int, error)
	AddAlert(alert *v1.Alert) error
	UpdateAlert(alert *v1.Alert) error
	MarkAlertStale(id string) error
}

// New returns a new soleInstance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
}
