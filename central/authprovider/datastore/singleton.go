package datastore

import (
	"github.com/stackrox/rox/central/authprovider/datastore/internal/store/bolt"
	pgStore "github.com/stackrox/rox/central/authprovider/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once         sync.Once
	soleInstance authproviders.Store
)

// Singleton returns the sole instance of the DataStore service.
func Singleton() authproviders.Store {
	once.Do(func() {
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			soleInstance = New(pgStore.New(globaldb.GetPostgres()))
		} else {
			soleInstance = New(bolt.New(globaldb.GetGlobalDB()))
		}
	})
	return soleInstance
}
