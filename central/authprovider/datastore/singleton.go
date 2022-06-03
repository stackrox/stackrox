package datastore

import (
	"github.com/stackrox/rox/central/authprovider/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/authprovider/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
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
