package datastore

import (
	pgStore "github.com/stackrox/rox/central/complianceoperator/rules/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once
	ds   DataStore
)

// Singleton returns the singleton datastore
func Singleton() DataStore {
	once.Do(func() {
		storage := pgStore.New(globaldb.GetPostgres())
		var err error
		ds, err = NewDatastore(storage)
		utils.CrashOnError(err)
	})
	return ds
}
