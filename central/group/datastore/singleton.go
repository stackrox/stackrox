package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/group/datastore/internal/store/postgres"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	ds                DataStore
	once              sync.Once
	authProviderMutex sync.RWMutex

	// Global variable to store the auth provider registry function
	// This will be set by the auth provider registry package to break circular dependency.
	authProviderRegistryFunc func() authproviders.Registry
)

func initialize() {
	ds = New(pgStore.New(globaldb.GetPostgres()), roleDatastore.Singleton(), func() authproviders.Registry {
		return concurrency.WithRLock1(&authProviderMutex, authProviderRegistryFunc)
	})

	// Give datastore access to groups so that it can delete any groups with empty props on startup
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	utils.Should(ds.RemoveAllWithEmptyProperties(ctx))
}

// SetAuthProviderRegistryFunc sets the function to get the auth provider registry
// This is called by the auth provider registry package to break the circular dependency
func SetAuthProviderRegistryFunc(fn func() authproviders.Registry) {
	concurrency.WithLock(&authProviderMutex, func() {
		authProviderRegistryFunc = fn
	})
}

// Singleton returns the singleton providing access to the roles store.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
