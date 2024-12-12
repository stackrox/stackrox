package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/user/datastore/internal/store"
)

// GetTestDataStore returns a datastore instance for testing purposes.
func GetTestDataStore(_ testing.TB) DataStore {
	testStorage := store.New()
	return New(testStorage)
}
