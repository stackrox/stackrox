package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	groupFilter "github.com/stackrox/rox/central/group/datastore/filter"
	"github.com/stackrox/rox/central/role/store"
	PermissionSetPGStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	postgresRolePGStore "github.com/stackrox/rox/central/role/store/role/postgres"
	postgresSimpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
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
		roleStorage := postgresRolePGStore.New(globaldb.GetPostgres())
		permissionSetStorage := PermissionSetPGStore.New(globaldb.GetPostgres())
		accessScopeStorage := postgresSimpleAccessScopeStore.New(globaldb.GetPostgres())
		// Which role format is used is determined solely by the feature flag.
		ds = New(roleStorage, permissionSetStorage, accessScopeStorage, groupFilter.GetFiltered)

		ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Access)))
		utils.Must(roleStorage.UpsertMany(ctx, getDefaultRoles()))
		utils.Must(seedDefaultPermissionSets(ctx, permissionSetStorage, getDefaultPermissionSets()))
		utils.Must(accessScopeStorage.UpsertMany(ctx, getDefaultAccessScopes()))
	})
	return ds
}

func seedDefaultPermissionSets(ctx context.Context, permissionSetStorage store.PermissionSetStore,
	defaults []*storage.PermissionSet) error {
	var consoleProxy *storage.PermissionSet
	others := make([]*storage.PermissionSet, 0, len(defaults))
	for _, permissionSet := range defaults {
		if permissionSet.GetName() == accesscontrol.K8sCentralProxyUser {
			consoleProxy = permissionSet
		} else {
			others = append(others, permissionSet)
		}
	}
	if len(others) > 0 {
		if err := permissionSetStorage.UpsertMany(ctx, others); err != nil {
			return err
		}
	}
	if consoleProxy != nil {
		if err := permissionSetStorage.Upsert(ctx, consoleProxy); err != nil {
			log.Errorf("Failed to seed Console Proxy permission set %q (ID %s); resolve any name conflict: %v",
				consoleProxy.GetName(), consoleProxy.GetId(), err)
		}
	}
	return nil
}
