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
					AddStrings(search.EventLevel, "info").
					ProtoQuery(),
			)
		})
		_ = phonehome.AddTotal(ctx, props, "Warning type administrative events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, "warn").
					ProtoQuery(),
			)
		})
		_ = phonehome.AddTotal(ctx, props, "Error type administrative events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, "error").
					ProtoQuery(),
			)
		})
		return props, nil
	}
}

/*

		return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_INFO
	case "warn":
		return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING
	case "error":
		return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR
	default:
		return storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_UNKNOWN

*/
