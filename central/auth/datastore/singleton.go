package datastore

import (
	"github.com/stackrox/rox/central/auth/m2m"
	pgStore "github.com/stackrox/rox/central/auth/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/jwt"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
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
		// However, we do this in the background since the creation of the token exchanger
		// will reach out to the OIDC provider's configuration endpoint.
		go func() {
			utils.Should(ds.(*datastoreImpl).InitializeTokenExchangers())
		}()
	})
	return ds
}
