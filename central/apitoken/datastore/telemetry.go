package datastore

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather the number of configured API tokens.
func Gather(ds DataStore) phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		ctx = sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Integration),
			),
		)
		numTotalTokens, err := ds.Count(ctx, &v1.Query{})
		if err != nil {
			return nil, errors.Wrap(err, "counting all api tokens")
		}

		formattedNow := time.Now().Format(TimestampLayout)
		numExpiredTokens, err := ds.Count(ctx,
			search.NewQueryBuilder().
				AddStrings(search.Expiration, fmt.Sprintf("<%s", formattedNow)).
				ProtoQuery(),
		)
		if err != nil {
			return nil, errors.Wrap(err, "counting expired api tokens")
		}

		numRevokedTokens, err := ds.Count(ctx,
			search.NewQueryBuilder().
				AddBools(search.Revoked, true).
				ProtoQuery(),
		)
		if err != nil {
			return nil, errors.Wrap(err, "counting revoked api tokens")
		}

		numValidTokens, err := ds.Count(ctx,
			search.NewQueryBuilder().
				AddBools(search.Revoked, false).
				AddStrings(search.Expiration, fmt.Sprintf(">%s", formattedNow)).
				ProtoQuery(),
		)
		if err != nil {
			return nil, errors.Wrap(err, "counting valid api tokens")
		}

		props := map[string]any{
			"Total API Tokens":         numTotalTokens,
			"Total API Tokens Expired": numExpiredTokens,
			"Total API Tokens Revoked": numRevokedTokens,
			"Total API Tokens Valid":   numValidTokens,
		}
		return props, nil
	}
}
