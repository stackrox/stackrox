//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGather(t *testing.T) {
	pool := pgtest.ForT(t)
	ds := GetTestPostgresDataStore(t, pool.DB)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)

	cloudSources := fixtures.GetManyStorageCloudSources(100)
	for _, cs := range cloudSources {
		require.NoError(t, ds.UpsertCloudSource(ctx, cs))
	}

	gatherFunc := Gather(ds)

	props, err := gatherFunc(ctx)
	require.NoError(t, err)

	expectedProps := map[string]any{
		"Total Cloud Sources":               100,
		"Total Paladin_cloud Cloud Sources": 50,
		"Total Ocm Cloud Sources":           50,
		"Total Unspecified Cloud Sources":   0,
	}

	assert.Equal(t, expectedProps, props)
}
