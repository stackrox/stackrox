package datastore

import (
	"github.com/stackrox/rox/central/administration/usage/store/postgres"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once

	log = logging.LoggerForModule()
)

// Singleton returns the singleton providing access to the administration usage store.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(postgres.New(globaldb.GetPostgres()), clusterDS.Singleton())
	})
	return ds
}
