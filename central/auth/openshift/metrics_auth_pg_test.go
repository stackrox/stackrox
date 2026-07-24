//go:build sql_integration

package openshift

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/globaldb"
	postgresSimpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedAccessScope(t *testing.T) {
	pgTest := pgtest.ForT(t)
	require.NotNil(t, pgTest)
	defer pgTest.Close()
	globaldb.SetPostgresTest(t, pgTest.DB)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	seedAccessScope(ctx)

	store := postgresSimpleAccessScopeStore.New(pgTest.DB)
	stored, found, err := store.Get(ctx, accessScopeID)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, accessScopeName, stored.GetName())
	assert.Equal(t, storage.Traits_DEFAULT, stored.GetTraits().GetOrigin())
	require.Len(t, stored.GetRules().GetClusterLabelSelectors(), 1)
	req := stored.GetRules().GetClusterLabelSelectors()[0].GetRequirements()[0]
	assert.Equal(t, centralServicesLabelKey, req.GetKey())
	assert.Equal(t, storage.SetBasedLabelSelector_EXISTS, req.GetOp())

	// Re-seeding is idempotent, matching the boot-time self-healing behavior.
	seedAccessScope(ctx)
	stored, found, err = store.Get(ctx, accessScopeID)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, accessScopeName, stored.GetName())
}
