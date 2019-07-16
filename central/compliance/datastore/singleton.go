package datastore

import (
	"github.com/stackrox/rox/central/compliance/datastore/internal/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once       sync.Once
	dsInstance DataStore
)

// Singleton returns the compliance DataStore singleton.
func Singleton() DataStore {
	once.Do(func() {
		boltStore, err := store.NewBoltStore(globaldb.GetGlobalDB())
		utils.Must(err)

		dsInstance = NewDataStore(boltStore, NewSacFilter())
	})
	return dsInstance
}
