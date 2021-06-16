package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	rolePkg "github.com/stackrox/rox/central/role"
	roleStore "github.com/stackrox/rox/central/role/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	permissionSetStore "github.com/stackrox/rox/central/role/store/permissionset/rocksdb"
	simpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/rocksdb"
	roleUtils "github.com/stackrox/rox/central/role/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
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
		roleStorage := roleStore.New(globaldb.GetGlobalDB())
		permissionSetStorage, err := permissionSetStore.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)
		accessScopeStorage, err := simpleAccessScopeStore.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)

		sacV2Enabled := features.ScopedAccessControl.Enabled()
		ds = New(roleStorage, permissionSetStorage, accessScopeStorage, sacV2Enabled)

		roles, permissionSets := getDefaultObjects()
		utils.Must(upsertDefaultRoles(roleStorage, roles))
		utils.Must(permissionSetStorage.UpsertMany(permissionSets))
	})
	return ds
}

type roleAttributes struct {
	idSuffix           string
	resourceWithAccess []permissions.ResourceWithAccess
}

var defaultRoles = map[string]roleAttributes{
	rolePkg.Admin: {
		idSuffix:           "admin",
		resourceWithAccess: resources.AllResourcesModifyPermissions(),
	},
	rolePkg.Analyst: {
		idSuffix:           "analyst",
		resourceWithAccess: resources.AllResourcesViewPermissions(),
	},
	rolePkg.ContinuousIntegration: {
		idSuffix: "continuousintegration",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Detection),
			permissions.Modify(resources.Image),
		},
	},
	rolePkg.None: {
		idSuffix: "none",
	},
	rolePkg.SensorCreator: {
		idSuffix: "sensorcreator",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Cluster),
			permissions.Modify(resources.Cluster),
			permissions.Modify(resources.ServiceIdentity),
		},
	},
}

func getDefaultObjects() ([]*storage.Role, []*storage.PermissionSet) {
	roles := make([]*storage.Role, 0, len(defaultRoles))
	permissionSets := make([]*storage.PermissionSet, 0, len(defaultRoles))

	for roleName, attributes := range defaultRoles {
		role := permissions.NewRoleWithAccess(roleName, attributes.resourceWithAccess...)
		role.Id = roleUtils.EnsureValidRoleID(attributes.idSuffix)

		permissionSet := &storage.PermissionSet{
			Id:               roleUtils.EnsureValidPermissionSetID(attributes.idSuffix),
			Name:             role.Name,
			Description:      role.Description,
			ResourceToAccess: role.ResourceToAccess,
		}
		role.PermissionSetId = permissionSet.Id

		roles = append(roles, role)
		permissionSets = append(permissionSets, permissionSet)
	}
	return roles, permissionSets
}

func upsertDefaultRoles(store roleStore.Store, roles []*storage.Role) error {
	// The default values may already exist in the Store, but we use
	// Add and Update in combination to "Upsert" to the latest set of
	// permissions. If *both* fail then we have an actual problem.
	for _, role := range roles {
		addRoleErr := store.AddRole(role)
		if addRoleErr != nil {
			updateErr := store.UpdateRole(role)
			if updateErr != nil {
				return errors.Wrapf(updateErr, "cannot upsert predefined role %s", role.Name)
			}
		}
	}
	return nil
}
