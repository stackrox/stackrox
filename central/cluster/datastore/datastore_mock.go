package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/cluster/store"
)

// MockDataStore is a mock implementation of the DataStore interface.
type MockDataStore struct {
	store.MockStore
}
