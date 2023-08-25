package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  DataStore
	once sync.Once
)

func initialize() {
	svc = NewPostgres(globaldb.GetPostgres())
}

// Singleton returns the API token singleton.
func Singleton() DataStore {
	once.Do(initialize)
	return svc
}
