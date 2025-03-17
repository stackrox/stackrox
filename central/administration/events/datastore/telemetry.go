package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

func Gather(ds DataStore) phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Integration),
			),
		)
		props := map[string]any{}
		_ = phonehome.AddTotal(ctx, props, "Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx, search.EmptyQuery())
		})
		_ = phonehome.AddTotal(ctx, props, "Info type administrative events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_INFO.String()).
					ProtoQuery(),
			)
		})
		_ = phonehome.AddTotal(ctx, props, "Warning type administrative events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING.String()).
					ProtoQuery(),
			)
		})
		_ = phonehome.AddTotal(ctx, props, "Error type administrative events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR.String()).
					ProtoQuery(),
			)
		})
		return props, nil
	}
}
