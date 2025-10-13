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
	return computeTelemetryProperties(ctx, integrations), nil
}

func computeTelemetryProperties(ctx context.Context, integrations []*storage.SignatureIntegration) map[string]any {
	if len(integrations) == 0 {
		return nil
	}

	totalPublicKeys, totalCertificates := 0, 0
	totalCertsWithCustomChain, totalCertsWithIntermediateCert := 0, 0
	totalCtlogEnabled, totalTlogEnabled, totalCustomRekorURL, totalValidateOffline := 0, 0, 0, 0
	for _, i := range integrations {
		totalPublicKeys += len(i.GetCosign().GetPublicKeys())
		totalCertificates += len(i.GetCosignCertificates())
		for _, cert := range i.GetCosignCertificates() {
			if len(cert.GetCertificateChainPemEnc()) > 0 {
				totalCertsWithCustomChain++
			}
			if len(cert.GetCertificatePemEnc()) > 0 {
				totalCertsWithIntermediateCert++
			}
			if cert.GetCertificateTransparencyLog().GetEnabled() {
				totalCtlogEnabled++
			}
		}
		if tlog := i.GetTransparencyLog(); tlog.GetEnabled() {
			totalTlogEnabled++
			if tlog.GetUrl() != "" && tlog.GetUrl() != "https://rekor.sigstore.dev" {
				totalCustomRekorURL++
			}
			if tlog.GetValidateOffline() {
				totalValidateOffline++
			}
		}
	}

	totals := make(map[string]any)
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integrations", phonehome.Len(integrations))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration Cosign Public Keys", phonehome.Constant(totalPublicKeys))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration Certificates", phonehome.Constant(totalCertificates))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration With Custom Certificate", phonehome.Constant(totalCertsWithIntermediateCert))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration With Custom Chain", phonehome.Constant(totalCertsWithCustomChain))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration With Certificate Transparency Log Validation", phonehome.Constant(totalCtlogEnabled))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration With Transparency Log Validation", phonehome.Constant(totalTlogEnabled))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration With Transparency Log Custom Rekor URL", phonehome.Constant(totalCustomRekorURL))
	_ = phonehome.AddTotal(ctx, totals,
		"Signature Integration With Transparency Log Offline Validation", phonehome.Constant(totalValidateOffline))
	return totals
}
