package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/blob/datastore/store"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
)

// NewTestDatastore creates the data store for testing and benchmarks.
func NewTestDatastore(_ testing.TB, db pgPkg.DB) Datastore {
	return &datastoreImpl{
		store: store.New(db),
	}
}
