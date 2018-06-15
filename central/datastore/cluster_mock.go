package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/db"
)

// MockClusterDataStore is a mock implementation of the ClusterDataStore interface.
type MockClusterDataStore struct {
	db.MockClusterStorage
}
