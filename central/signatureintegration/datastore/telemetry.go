package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather number of signature integrations.
func Gather(ctx context.Context) (phonehome.Properties, error) {
	// WithAllAccess is required only to fetch and calculate the number of
	// signature integrations. It is not propagated anywhere else.
	ctx = sac.WithAllAccess(ctx)
	totals := make(phonehome.Properties)
	ps, err := Singleton().GetAllSignatureIntegrations(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signature integrations")
	}
	totals["Total Signature Integrations"] = len(ps)
	return totals, nil
}
