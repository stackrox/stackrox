package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errorhelpers"
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
		errorList := errorhelpers.NewErrorList("Administrative Events Telemetry")
		err := phonehome.AddTotal(ctx, props, "Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx, search.EmptyQuery())
		})
		errorList.AddError(err)
		err = phonehome.AddTotal(ctx, props, "Info type Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_INFO.String()).
					ProtoQuery(),
			)
		})
		errorList.AddError(err)
		err = phonehome.AddTotal(ctx, props, "Warning type Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING.String()).
					ProtoQuery(),
			)
		})
		errorList.AddError(err)
		err = phonehome.AddTotal(ctx, props, "Error type Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR.String()).
					ProtoQuery(),
			)
		})
		errorList.AddError(err)
		err = phonehome.AddTotal(ctx, props, "Authentication domain Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventDomain, events.AuthenticationDomain).
					ProtoQuery(),
			)
		})
		err = phonehome.AddTotal(ctx, props, "Default domain Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventDomain, events.DefaultDomain).
					ProtoQuery(),
			)
		})
		err = phonehome.AddTotal(ctx, props, "Image Scanning domain Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventDomain, events.ImageScanningDomain).
					ProtoQuery(),
			)
		})
		err = phonehome.AddTotal(ctx, props, "Integration domain Administrative Events", func(ctx context.Context) (int, error) {
			return ds.CountEvents(ctx,
				search.NewQueryBuilder().
					AddStrings(search.EventDomain, events.IntegrationDomain).
					ProtoQuery(),
			)
		})
		errorList.AddError(err)
		return props, errorList.ToError()
	}
}
