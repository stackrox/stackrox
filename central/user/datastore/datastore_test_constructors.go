package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/user/datastore/internal/store"
)

// GetTestDatastore returns a datastore instance for testing purposes.
func GetTestDatastore(_ testing.TB) DataStore {
	testStorage := store.New()
	return New(testStorage)
}
