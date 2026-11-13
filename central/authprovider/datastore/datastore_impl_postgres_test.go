//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/require"
)

func TestAuthProviderExistsWithNamePostgres(t *testing.T) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	testPostgres := pgtest.ForT(t)
	store := GetTestPostgresDataStore(t, testPostgres.DB)

	exists, err := store.AuthProviderExistsWithName(ctx, "provider")
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, store.AddAuthProvider(ctx, &storage.AuthProvider{
		Id:       "id",
		Name:     "provider",
		LoginUrl: "login-url",
	}))

	exists, err = store.AuthProviderExistsWithName(ctx, "provider")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = store.AuthProviderExistsWithName(ctx, "other-provider")
	require.NoError(t, err)
	require.False(t, exists)
}
