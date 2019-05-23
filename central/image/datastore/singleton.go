package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var err error
	ad, err = New(globaldb.GetGlobalDB(), globalindex.GetGlobalIndex(), false)
	if err != nil {
		panic(errors.Wrap(err, "could not create images datastore"))
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
