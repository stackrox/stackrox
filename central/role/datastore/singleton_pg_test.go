//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	rolePkg "github.com/stackrox/rox/central/role"
	permissionSetPGStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedProxyPermissionSetNameCollision(t *testing.T) {
	for name, seed := range map[string]func(context.Context, permissionSetPGStore.Store) error{
		accesscontrol.ConsoleProxy: func(ctx context.Context, store permissionSetPGStore.Store) error {
			return seedDefaultPermissionSets(ctx, store,
				[]*storage.PermissionSet{getDefaultPermissionSet(accesscontrol.ConsoleProxy)})
		},
		accesscontrol.OpenShiftPrometheusMetricsReader: func(ctx context.Context, store permissionSetPGStore.Store) error {
			seedMetricsPermissionSet(ctx, store)
			return nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			pgTest := pgtest.ForT(t)
			defer pgTest.Close()

			ctx := sac.WithAllAccess(t.Context())
			store := permissionSetPGStore.New(pgTest.DB)
			custom := &storage.PermissionSet{
				Id:               rolePkg.GeneratePermissionSetID(),
				Name:             name,
				ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_WRITE_ACCESS},
			}
			require.NoError(t, store.Upsert(ctx, custom))

			for range 2 {
				require.NoError(t, seed(ctx, store))
			}
			stored, found, err := store.Get(ctx, custom.GetId())
			require.NoError(t, err)
			require.True(t, found)
			protoassert.Equal(t, custom, stored)
			assert.False(t, rolePkg.IsDefaultPermissionSet(stored))
			count, err := store.Count(ctx, search.EmptyQuery())
			require.NoError(t, err)
			assert.Equal(t, 1, count)
		})
	}
}

func TestSeedMetricsPermissionSetPreservesAdminGrants(t *testing.T) {
	for name, grants := range map[string]map[string]storage.Access{
		"empty":       {},
		"scoped read": {"Deployment": storage.Access_READ_ACCESS},
		"write":       {"Deployment": storage.Access_READ_WRITE_ACCESS},
		"global read": {"Administration": storage.Access_READ_ACCESS},
	} {
		t.Run(name, func(t *testing.T) {
			pgTest := pgtest.ForT(t)
			defer pgTest.Close()

			ctx := sac.WithAllAccess(t.Context())
			store := permissionSetPGStore.New(pgTest.DB)
			existing := &storage.PermissionSet{
				Id:               accesscontrol.OpenShiftPrometheusMetricsReaderID,
				Name:             "Administrator renamed metrics",
				Description:      "Administrator description",
				ResourceToAccess: grants,
			}
			require.NoError(t, store.Upsert(ctx, existing))
			for range 2 {
				seedMetricsPermissionSet(ctx, store)
			}
			stored, found, err := store.Get(ctx, existing.GetId())
			require.NoError(t, err)
			require.True(t, found)
			protoassert.Equal(t, existing, stored)
			require.False(t, stored.GetAccessConstraint() == storage.AccessConstraint_ACCESS_CONSTRAINT_READ &&
				stored.GetScopeConstraint() == storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
				"existing grants need explicit constraints before use for scoped tokens")
		})
	}
}
