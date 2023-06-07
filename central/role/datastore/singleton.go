package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	groupFilter "github.com/stackrox/rox/central/group/datastore/filter"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/role/store"
	PermissionSetPGStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	permissionSetPGStore "github.com/stackrox/rox/central/role/store/permissionset/rocksdb"
	postgresRolePGStore "github.com/stackrox/rox/central/role/store/role/postgres"
	roleStore "github.com/stackrox/rox/central/role/store/role/rocksdb"
	postgresSimpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	simpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton providing access to the roles store.
func Singleton() DataStore {
	once.Do(func() {
		var roleStorage store.RoleStore
		var permissionSetStorage store.PermissionSetStore
		var accessScopeStorage store.SimpleAccessScopeStore
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			roleStorage = postgresRolePGStore.New(globaldb.GetPostgres())
			permissionSetStorage = PermissionSetPGStore.New(globaldb.GetPostgres())
			accessScopeStorage = postgresSimpleAccessScopeStore.New(globaldb.GetPostgres())
		} else {
			var err error
			roleStorage, err = roleStore.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
			permissionSetStorage, err = permissionSetPGStore.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
			accessScopeStorage, err = simpleAccessScopeStore.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
		}
		// Which role format is used is determined solely by the feature flag.
		ds = New(roleStorage, permissionSetStorage, accessScopeStorage, groupFilter.GetFiltered)

		ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Access)))
		roles, permissionSets, accessScopes := getDefaultObjects()
		utils.Must(roleStorage.UpsertMany(ctx, roles))
		utils.Must(permissionSetStorage.UpsertMany(ctx, permissionSets))
		utils.Must(accessScopeStorage.UpsertMany(ctx, accessScopes))
	})
	return ds
}

func getDefaultObjects() ([]*storage.Role, []*storage.PermissionSet, []*storage.SimpleAccessScope) {
	return getDefaultRoles(), getDefaultPermissionSets(), getDefaultAccessScopes()
}
