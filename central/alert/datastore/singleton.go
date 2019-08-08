package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/central/alert/datastore/internal/store/badger"
	"github.com/stackrox/rox/central/alert/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once         sync.Once
	soleInstance DataStore
)

func initialize() {
	var storage store.Store
	if features.BadgerDB.Enabled() {
		storage = badger.New(globaldb.GetGlobalBadgerDB())
	} else {
		storage = bolt.New(globaldb.GetGlobalDB())
	}
	indexer := index.New(globalindex.GetGlobalIndex())
	searcher := search.New(storage, indexer)

	var err error
	soleInstance, err = New(storage, indexer, searcher)
	utils.Must(errors.Wrap(err, "unable to load datastore for alerts"))
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(initialize)
	return soleInstance
}
