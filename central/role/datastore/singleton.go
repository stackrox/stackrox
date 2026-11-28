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
		seedMetricsPermissionSet(ctx, permissionSetStorage)
	})
	return ds
}

func seedDefaultPermissionSets(ctx context.Context, permissionSetStorage store.PermissionSetStore,
	defaults []*storage.PermissionSet) error {
	var k8sCentralProxyUser *storage.PermissionSet
	others := make([]*storage.PermissionSet, 0, len(defaults))
	for _, permissionSet := range defaults {
		if permissionSet.GetName() == accesscontrol.K8sCentralProxyUser {
			k8sCentralProxyUser = permissionSet
		} else {
			others = append(others, permissionSet)
		}
	}
	if len(others) > 0 {
		if err := permissionSetStorage.UpsertMany(ctx, others); err != nil {
			return err
		}
	}
	if k8sCentralProxyUser != nil {
		if err := permissionSetStorage.Upsert(ctx, k8sCentralProxyUser); err != nil {
			log.Errorf("Failed to seed default permission set %q (ID %s); resolve any name conflict: %v",
				k8sCentralProxyUser.GetName(), k8sCentralProxyUser.GetId(), err)
		}
	}
	return nil
}

// seedMetricsPermissionSet provisions metrics permissions on every Central, including
// those managing OpenShift secured clusters from a non-OpenShift management cluster.
// Existing administrator-managed grants and constraints are preserved. Before using
// an existing set for scoped tokens, administrators must explicitly configure READ
// and CLUSTER_NAMESPACE constraints and remove any incompatible grants.
// Only the original seeded name is updated; administrator-chosen names are preserved.
func seedMetricsPermissionSet(ctx context.Context, permissionSetStorage store.PermissionSetStore) {
	permissionSet, exists, err := permissionSetStorage.Get(ctx, accesscontrol.OpenShiftPrometheusID)
	if err != nil {
		log.Warnf("Failed to get metrics permission set: %v", err)
		return
	}
	if exists {
		if permissionSet.GetName() != "OpenShift Prometheus Metrics Reader" {
			return
		}
		permissionSet = permissionSet.CloneVT()
		permissionSet.Name = accesscontrol.OpenShiftPrometheus
	} else {
		permissionSet = &storage.PermissionSet{
			Id:               accesscontrol.OpenShiftPrometheusID,
			Name:             accesscontrol.OpenShiftPrometheus,
			Description:      "For OpenShift Prometheus: use it to grant read-only access to scoped resources used by custom metrics",
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			// Grant no permissions until an administrator chooses which metrics to expose.
			ResourceToAccess: map[string]storage.Access{},
			Traits:           &storage.Traits{Origin: storage.Traits_IMPERATIVE},
		}
	}
	if err := permissionSetStorage.Upsert(ctx, permissionSet); err != nil {
		log.Warnf("Failed to seed metrics permission set %q (ID %s); resolve any name conflict: %v",
			permissionSet.GetName(), permissionSet.GetId(), err)
	}
}
