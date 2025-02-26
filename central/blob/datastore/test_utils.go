package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/blob/datastore/search"
	"github.com/stackrox/rox/central/blob/datastore/store"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
)

// NewTestDatastore creates the data store for testing.
func NewTestDatastore(_ *testing.T, db pgPkg.DB) Datastore {
	datastore := store.New(db)
	searcher := search.New(datastore)
	return &datastoreImpl{
		store:    datastore,
		searcher: searcher,
	}
}
