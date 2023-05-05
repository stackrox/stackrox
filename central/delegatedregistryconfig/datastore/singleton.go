package datastore

import (
	pgStore "github.com/stackrox/rox/central/delegatedregistryconfig/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	DefaultEnabledFor = storage.DelegatedRegistryConfig_NONE
)

var (
	once sync.Once

	d DataStore

	defaultConfig = storage.DelegatedRegistryConfig{
		EnabledFor: DefaultEnabledFor,
	}
)

func initialize() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		// only postgres supported for this datastore
		return
	}

	d = New(pgStore.New(globaldb.GetPostgres()))

	// ctx := sac.WithGlobalAccessScopeChecker(
	// 	context.Background(),
	// 	sac.AllowFixedScopes(
	// 		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
	// 		sac.ResourceScopeKeys(resources.Administration)))

	// config, err := d.GetConfig(ctx)
	// utils.CrashOnError(err)

	// if config != nil {
	// 	return
	// }

	// utils.Must(d.UpsertConfig(ctx, &defaultConfig))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return d
}
