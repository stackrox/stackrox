package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/hash/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds Datastore
)

// Singleton returns the singleton Hash datastore
func Singleton() Datastore {
	once.Do(func() {
		ds = NewDatastore(postgres.New(globaldb.GetPostgres()))
	})
	return ds
}
