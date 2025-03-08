//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/delegatedregistryconfig/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGather(t *testing.T) {
	pool := pgtest.ForT(t)
	ds := New(pgStore.New(pool.DB))
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)

	testCases := map[string]struct {
		config                  *storage.DelegatedRegistryConfig
		registriesWithClusterID int
	}{
		"enabled for all": {
			config: &storage.DelegatedRegistryConfig{
				EnabledFor:       storage.DelegatedRegistryConfig_ALL,
				DefaultClusterId: "my-cluster",
				Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
					{Path: "quay.io/rhacs-eng/qa"},
				},
			},
		},
		"enabled for none": {
			config: &storage.DelegatedRegistryConfig{
				EnabledFor: storage.DelegatedRegistryConfig_NONE,
				Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{},
			},
		},
		"enabled for specific": {
			config: &storage.DelegatedRegistryConfig{
				EnabledFor:       storage.DelegatedRegistryConfig_SPECIFIC,
				DefaultClusterId: "my-cluster",
				Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
					{Path: "quay.io/rhacs-eng/qa", ClusterId: "my-cluster"},
					{Path: "quay.io/rhacs-eng/main"},
				},
			},
			registriesWithClusterID: 1,
		},
		"default values when no config is found": {},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := ds.UpsertConfig(ctx, tc.config)
			require.NoError(t, err)

			gatherFunc := Gather(ds)
			props, err := gatherFunc(ctx)
			require.NoError(t, err)

			expectedProps := map[string]any{
				"Delegated Scanning Config Enabled For":                           tc.config.GetEnabledFor().String(),
				"Delegated Scanning Config Default Cluster ID Populated":          tc.config.GetDefaultClusterId() != "",
				"Total Delegated Scanning Config Registries":                      len(tc.config.GetRegistries()),
				"Total Delegated Scanning Config Registries Cluster ID Populated": tc.registriesWithClusterID,
			}
			assert.Equal(t, expectedProps, props)
		})
	}
}
