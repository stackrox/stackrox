package datastore

import (
	"github.com/stackrox/rox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// New returns a new Store instance.
func New(storage store.Store) authproviders.Store {
	return &datastoreImpl{
		storage: storage,
	}
}
