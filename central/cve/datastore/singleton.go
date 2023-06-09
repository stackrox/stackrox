package datastore

import (
	"github.com/stackrox/rox/central/cve/store/dackbox"
	globaldb "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	globalDackbox := globaldb.GetGlobalDackBox()

	storage := dackbox.New(globalDackbox, globaldb.GetKeyFence())
	var err error
	ds, err = New(globalDackbox, globaldb.GetIndexQueue(), storage, nil, nil)
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
