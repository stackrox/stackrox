package datastore

import (
	"github.com/stackrox/rox/central/alert/index"
	"github.com/stackrox/rox/central/alert/search"
	"github.com/stackrox/rox/central/alert/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is a transaction script with methods that provide the domain logic for CRUD uses cases for Alert objects.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(q *v1.Query) ([]searchPkg.Result, error)
	SearchAlerts(q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawAlerts(q *v1.Query) ([]*storage.Alert, error)
	SearchListAlerts(q *v1.Query) ([]*storage.ListAlert, error)

	ListAlerts(request *v1.ListAlertsRequest) ([]*storage.ListAlert, error)

	GetAlertStore() ([]*storage.ListAlert, error)
	GetAlert(id string) (*storage.Alert, bool, error)
	CountAlerts() (int, error)
	AddAlert(alert *storage.Alert) error
	UpdateAlert(alert *storage.Alert) error
	MarkAlertStale(id string) error
}

// New returns a new soleInstance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:    storage,
		indexer:    indexer,
		searcher:   searcher,
		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
}
