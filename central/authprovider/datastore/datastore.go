package datastore

import (
	"github.com/stackrox/stackrox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/stackrox/pkg/auth/authproviders"
)

// New returns a new Store instance using the provided bolt DB instance.
func New(storage store.Store) authproviders.Store {
	return &datastoreImpl{
		storage: storage,
	}
}
