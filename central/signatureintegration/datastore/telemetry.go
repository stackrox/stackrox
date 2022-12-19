package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather number of signature integrations.
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	// WithAllAccess is required only to fetch and calculate the number of
	// signature integrations. It is not propagated anywhere else.
	ctx = sac.WithAllAccess(ctx)
	totals := make(map[string]any)
	si := Singleton()
	if err := phonehome.AddTotal(ctx, totals, "Signature Integrations", si.GetAllSignatureIntegrations); err != nil {
	        return nil, errors.Wrap(err, "failed to get signature integrations")
	}
	return totals, nil
	return totals, nil
}
