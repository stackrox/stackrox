package datastore

import (
	"context"
	"errors"

	"github.com/stackrox/rox/central/globaldb"
	groupFilter "github.com/stackrox/rox/central/group/datastore/filter"
	"github.com/stackrox/rox/central/role/store"
	PermissionSetPGStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	postgresRolePGStore "github.com/stackrox/rox/central/role/store/role/postgres"
	postgresSimpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
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
		roles, permissionSets, accessScopes := getDefaultObjects()
		utils.Must(roleStorage.UpsertMany(ctx, roles))
		utils.Must(seedDefaultPermissionSets(ctx, ds, permissionSetStorage, permissionSets))
		utils.Must(accessScopeStorage.UpsertMany(ctx, accessScopes))
	})
	return ds
}

func seedDefaultPermissionSets(ctx context.Context, roleDS DataStore, permissionSetStorage store.PermissionSetStore, defaults []*storage.PermissionSet) error {
	var consoleProxy *storage.PermissionSet
	others := make([]*storage.PermissionSet, 0, len(defaults))
	for _, permissionSet := range defaults {
		if permissionSet.GetName() == accesscontrol.ConsoleProxy {
			consoleProxy = permissionSet
		} else {
			others = append(others, permissionSet)
		}
	}
	if err := permissionSetStorage.UpsertMany(ctx, others); err != nil {
		return err
	}
	if consoleProxy == nil {
		return nil
	}
	existing, err := consoleProxyWithName(ctx, roleDS, consoleProxy.GetName())
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.GetTraits().GetOrigin() != storage.Traits_DEFAULT {
			log.Warnf("Keeping existing permission set %q instead of seeding the Console Proxy default", consoleProxy.GetName())
			return nil
		}
		consoleProxy.Id = existing.GetId()
	}
	err = permissionSetStorage.Upsert(ctx, consoleProxy)
	if !errors.Is(err, errox.AlreadyExists) {
		return err
	}
	existing, lookupErr := consoleProxyWithName(ctx, roleDS, consoleProxy.GetName())
	if lookupErr != nil {
		return lookupErr
	}
	if existing != nil && existing.GetTraits().GetOrigin() != storage.Traits_DEFAULT {
		log.Warnf("Keeping existing permission set %q instead of seeding the Console Proxy default", consoleProxy.GetName())
		return nil
	}
	if existing != nil && existing.GetId() != consoleProxy.GetId() {
		consoleProxy.Id = existing.GetId()
		return permissionSetStorage.Upsert(ctx, consoleProxy)
	}
	return err
}

func consoleProxyWithName(ctx context.Context, roleDS DataStore, name string) (*storage.PermissionSet, error) {
	existing, err := roleDS.GetPermissionSetsFiltered(ctx, func(permissionSet *storage.PermissionSet) bool {
		return permissionSet.GetName() == name
	})
	if err != nil {
		return nil, err
	}
	if len(existing) == 0 {
		return nil, nil
	}
	return existing[0], nil
}

func getDefaultObjects() ([]*storage.Role, []*storage.PermissionSet, []*storage.SimpleAccessScope) {
	return getDefaultRoles(), getDefaultPermissionSets(), getDefaultAccessScopes()
}
