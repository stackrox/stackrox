package datastore

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	propertyName = "Cloud Sources"
)

func trimPrefix(typeName string) string {
	return strings.TrimPrefix(typeName, "TYPE_")
}

// Gather cloud source types.
// Current properties we gather:
// "Total Cloud Sources"
// "Total <cloud source type> Cloud Sources"
func Gather(ds DataStore) phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Integration),
			),
		)
		props := make(map[string]any)

		cloudSources, err := ds.ListCloudSources(ctx, search.EmptyQuery())
		if err != nil {
			return nil, errors.Wrap(err, "failed to get image integrations")
		}

		// Can safely ignore the error here since we already fetched integrations.
		_ = phonehome.AddTotal(ctx, props, propertyName, phonehome.Len(cloudSources))

		totalCount := map[string]int{}
		for csType := range storage.CloudSource_Type_value {
			totalCount[trimPrefix(csType)] = 0
		}

		for _, cs := range cloudSources {
			totalCount[trimPrefix(cs.GetType().String())]++
		}

		titleCase := cases.Title(language.English, cases.Compact).String

		for csType, count := range totalCount {
			props[fmt.Sprintf("Total %s %s",
				titleCase(csType), propertyName)] = count
		}

		return props, nil
	}
}
