package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once         sync.Once
	soleInstance DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	searcher := search.New(storage)
	var err error
	soleInstance, err = New(storage, searcher, platformmatcher.Singleton())
	utils.CrashOnError(errors.Wrap(err, "unable to load datastore for alerts"))
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(initialize)
	return soleInstance
}
