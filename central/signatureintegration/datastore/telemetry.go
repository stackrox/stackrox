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

	integrations, err := Singleton().GetAllSignatureIntegrations(ctx)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get all signature integrations")
	}
	if len(integrations) == 0 {
		return nil, nil
	}

	totalPublicKeys, totalCertificates := 0, 0
	for _, i := range integrations {
		totalPublicKeys += len(i.GetCosign().GetPublicKeys())
		totalCertificates += len(i.GetCosignCertificates())
	}

	totals := make(map[string]any)
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integrations", phonehome.Len(integrations))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration Cosign Public Keys", phonehome.Constant(totalPublicKeys))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration Certificates", phonehome.Constant(totalCertificates))
	return totals, nil
}
