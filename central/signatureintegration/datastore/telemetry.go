package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var log = logging.LoggerForModule()

var Gather phonehome.GatherFunc = func(ctx context.Context) (phonehome.Properties, error) {
	ctx = sac.WithAllAccess(ctx)
	totals := make(phonehome.Properties)
	if ps, err := Singleton().GetAllSignatureIntegrations(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to get signature integrations")
	} else {
		totals["Total Signature Integrations"] = len(ps)
	}
	return totals, nil
}
