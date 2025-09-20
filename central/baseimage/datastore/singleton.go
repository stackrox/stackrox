package datastore

import (
	baseImageStore "github.com/stackrox/rox/central/baseimage/store/postgres"
	baseImageLayerStore "github.com/stackrox/rox/central/baseimagelayer/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	db := globaldb.GetPostgres()
	baseImageStore := baseImageStore.New(db)
	baseImageLayerStore := baseImageLayerStore.New(db)
	ad = New(db, baseImageStore, baseImageLayerStore)
}

func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
