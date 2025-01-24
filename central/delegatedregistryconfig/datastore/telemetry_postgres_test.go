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
		enabledFor storage.DelegatedRegistryConfig_EnabledFor
		registries []*storage.DelegatedRegistryConfig_DelegatedRegistry
	}{
		"enabled for all": {
			enabledFor: storage.DelegatedRegistryConfig_ALL,
			registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "quay.io/rhacs-eng/qa"},
			},
		},
		"enabled for none": {
			enabledFor: storage.DelegatedRegistryConfig_NONE,
			registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{},
		},
		"enabled for specific": {
			enabledFor: storage.DelegatedRegistryConfig_SPECIFIC,
			registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "quay.io/rhacs-eng/qa"},
				{Path: "quay.io/rhacs-eng/main"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := ds.UpsertConfig(ctx,
				&storage.DelegatedRegistryConfig{
					EnabledFor: tc.enabledFor,
					Registries: tc.registries,
				},
			)
			require.NoError(t, err)

			gatherFunc := Gather(ds)
			props, err := gatherFunc(ctx)
			require.NoError(t, err)

			expectedProps := map[string]any{
				"Delegated Scanning Enabled For": tc.enabledFor.String(),
				"Total Delegated Registries":     len(tc.registries),
			}
			assert.Equal(t, expectedProps, props)
		})
	}
}
