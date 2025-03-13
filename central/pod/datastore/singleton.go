package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processindicator/filter"
	plopDS "github.com/stackrox/rox/central/processlisteningonport/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ps DataStore
)

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(func() {
		ps = NewPostgresDB(globaldb.GetPostgres(), piDS.Singleton(), plopDS.Singleton(), filter.Singleton())
	})
	return ps
}
