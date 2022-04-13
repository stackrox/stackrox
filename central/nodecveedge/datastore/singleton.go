package datastore

import (
	globaldb "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/nodecveedge/store/dackbox"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	ad = New(globaldb.GetGlobalDackBox(), dackbox.New(globaldb.GetGlobalDackBox()))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
