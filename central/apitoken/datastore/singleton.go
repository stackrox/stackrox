package datastore

import (
	"github.com/stackrox/rox/central/apitoken/datastore/internal/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  DataStore
	once sync.Once
)

func initialize() {
	svc = New(store.New(globaldb.GetGlobalDB()))
}

// Singleton returns the API token singleton.
func Singleton() DataStore {
	once.Do(initialize)
	return svc
}
