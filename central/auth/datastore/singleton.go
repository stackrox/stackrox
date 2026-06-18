package datastore

import (
	"errors"
	"os"

	"github.com/stackrox/rox/central/auth/m2m"
	"github.com/stackrox/rox/central/auth/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/jwt"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()

	once sync.Once

	ds DataStore
)

// Singleton provides a singleton auth machine to machine DataStore.
func Singleton() DataStore {
	once.Do(func() {
		set := m2m.TokenExchangerSetSingleton(roleDataStore.Singleton(), jwt.IssuerFactorySingleton())
		ds = New(
			store.New(globaldb.GetPostgres()),
			roleDataStore.Singleton(),
			set,
		)

		// On initialization of the store, list all existing configs and fill the set.
		// However, we do this in the background since the creation of the token exchanger
		// will reach out to the OIDC provider's configuration endpoint.
		go func() {
			if err := ds.InitializeTokenExchangers(); err != nil {
				if !buildinfo.ReleaseBuild && errors.Is(err, os.ErrNotExist) {
					log.Warnf("Skipping M2M token exchanger initialization (no Kubernetes service account token): %v", err)
					return
				}
				utils.Should(err)
			}
		}()
	})
	return ds
}
