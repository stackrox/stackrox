package datastore

import (
	"context"

	"github.com/stackrox/rox/central/auth/m2m"
	pgStore "github.com/stackrox/rox/central/auth/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/jwt"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

// Singleton provides a singleton auth machine to machine DataStore.
func Singleton() DataStore {
	once.Do(func() {
		set := m2m.TokenExchangerSetSingleton(roleDataStore.Singleton(), jwt.IssuerFactorySingleton())
		ds = New(pgStore.New(globaldb.GetPostgres()), set)

		// On initialization of the store, list all existing configs and fill the set.
		ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Access)))
		configs, err := ds.ListAuthM2MConfigs(ctx)
		utils.Must(err)
		for _, config := range configs {
			utils.Must(set.UpsertTokenExchanger(ctx, config))
		}
	})
	return ds
}
