package datastore

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather the number of administrative events
func Gather(ds DataStore) phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Administration),
			),
		)
		props := map[string]any{}
		var combinedErr error
		err := phonehome.AddTotal(ctx, props, "Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx, search.EmptyQuery())
		})
		if err != nil {
			combinedErr = multierror.Append(combinedErr, err)
		}
		err = phonehome.AddTotal(ctx, props, "Info type Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_INFO.String()).
					ProtoQuery(),
			)
		})
		if err != nil {
			combinedErr = multierror.Append(combinedErr, err)
		}
		err = phonehome.AddTotal(ctx, props, "Warning type Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING.String()).
					ProtoQuery(),
			)
		})
		if err != nil {
			combinedErr = multierror.Append(combinedErr, err)
		}
		err = phonehome.AddTotal(ctx, props, "Error type Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR.String()).
					ProtoQuery(),
			)
		})
		if err != nil {
			combinedErr = multierror.Append(combinedErr, err)
		}
		return props, combinedErr
	}
}
