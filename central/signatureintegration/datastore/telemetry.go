package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather number of signature integrations.
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	totals := make(map[string]any)
	si := Singleton()
	if err := phonehome.AddTotal(ctx, totals, "Signature Integrations", si.CountSignatureIntegrations); err != nil {
		return nil, errors.Wrap(err, "failed to get signature integrations")
	}
	return totals, nil
}
