package repository

import (
	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log  = logging.LoggerForModule()
	once sync.Once
	ds   DataStore
)

// Singleton returns the global datastore instance for base image repositories.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(repoStore.New(globaldb.GetPostgres()))
		log.Info("Initialized base image repository datastore with PostgreSQL backend")
	})
	return ds
}
