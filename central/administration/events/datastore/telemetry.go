package datastore

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	telemetryMap = map[string]*v1.Query{
		"Administration Events":                       search.EmptyQuery(),
		"Info type Administration Events":             stringQuery(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_INFO.String()),
		"Warning type Administration Events":          stringQuery(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING.String()),
		"Error type Administration Events":            stringQuery(search.EventLevel, storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR.String()),
		"Authentication domain Administration Events": stringQuery(search.EventDomain, events.AuthenticationDomain),
		"Default domain Administration Events":        stringQuery(search.EventDomain, events.DefaultDomain),
		"Image Scanning domain Administration Events": stringQuery(search.EventDomain, events.ImageScanningDomain),
		"Integration domain Administration Events":    stringQuery(search.EventDomain, events.IntegrationDomain),
	}
)

// Gather the number of administration events.
func Gather(ds DataStore) phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Administration),
			),
		)
		props := map[string]any{}
		errorList := errorhelpers.NewErrorList("Administration Events Telemetry")
		dsCountEvents := phonehome.Bind2nd(ds.CountEvents)
		for key, query := range telemetryMap {
			errorList.AddError(phonehome.AddTotal(ctx, props, key, dsCountEvents(query)))
		}
		return props, errorList.ToError()
	}
}

func stringQuery(label search.FieldLabel, value string) *v1.Query {
	return search.NewQueryBuilder().AddStrings(label, value).ProtoQuery()
}
