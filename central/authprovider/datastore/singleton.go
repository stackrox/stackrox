package datastore

import (
	"github.com/stackrox/stackrox/central/authprovider/datastore/internal/store/bolt"
	"github.com/stackrox/stackrox/central/authprovider/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/pkg/auth/authproviders"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once         sync.Once
	soleInstance authproviders.Store
)

// Singleton returns the sole instance of the DataStore service.
func Singleton() authproviders.Store {
	once.Do(func() {
		if features.PostgresDatastore.Enabled() {
			soleInstance = New(postgres.New(globaldb.GetPostgres()))
		} else {
			soleInstance = New(bolt.New(globaldb.GetGlobalDB()))
		}
	})
	return soleInstance
}
