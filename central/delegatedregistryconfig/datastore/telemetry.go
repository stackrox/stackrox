package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather the enabled for config and number of delegated registries.
func Gather(ds DataStore) phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Administration),
			),
		)
		cfg, _, err := ds.GetConfig(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get delegated registry config")
		}
		props := map[string]any{
			"Delegated Scanning Config Enabled For":                  cfg.GetEnabledFor().String(),
			"Delegated Scanning Config Default Cluster ID Populated": cfg.GetDefaultClusterId() != "",
		}
		_ = phonehome.AddTotal(ctx, props, "Delegated Scanning Config Registries", func(ctx context.Context) (int, error) {
			return len(cfg.GetRegistries()), nil
		})
		numRegistriesWithCluster := 0
		for _, reg := range cfg.GetRegistries() {
			if reg.GetClusterId() != "" {
				numRegistriesWithCluster++
			}
		}
		_ = phonehome.AddTotal(ctx, props, "Delegated Scanning Config Registries Cluster ID Populated", func(ctx context.Context) (int, error) {
			return numRegistriesWithCluster, nil
		})
		return props, nil
	}
}
