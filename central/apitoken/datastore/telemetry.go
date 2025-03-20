package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	revokedQuery = search.NewQueryBuilder().
		AddBools(search.Revoked, true).
		ProtoQuery()
)

// Gather the number of configured API tokens.
func Gather(ds DataStore) phonehome.GatherFunc {
	dsCount := phonehome.Bind2nd(ds.Count)

	return func(ctx context.Context) (map[string]any, error) {
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Integration),
			),
		)

		props := map[string]any{}
		_ = phonehome.AddTotal(ctx, props, "API Tokens", dsCount(search.EmptyQuery()))

		formattedNow := time.Now().Format(TimestampLayout)
		_ = phonehome.AddTotal(ctx, props, "API Tokens Expired", dsCount(
			search.NewQueryBuilder().
				AddStrings(search.Expiration, "<"+formattedNow).
				ProtoQuery()))

		_ = phonehome.AddTotal(ctx, props, "API Tokens Revoked", dsCount(revokedQuery))

		_ = phonehome.AddTotal(ctx, props, "API Tokens Valid", dsCount(
			search.NewQueryBuilder().
				AddBools(search.Revoked, false).
				AddStrings(search.Expiration, ">"+formattedNow).
				ProtoQuery()))
		return props, nil
	}
}
