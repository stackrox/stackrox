package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  DataStore
	once sync.Once
)

func initialize() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		svc = NewPostgres(globaldb.GetPostgres())
	} else {
		svc = NewRocks(globaldb.GetRocksDB())
	}
}

// Singleton returns the API token singleton.
func Singleton() DataStore {
	once.Do(initialize)
	return svc
}
