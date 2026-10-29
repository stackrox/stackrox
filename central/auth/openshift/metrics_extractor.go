package openshift

import (
	"context"
	"slices"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

type metricsExtractor struct {
	authn.IdentityExtractor
}

// GuardMetricsIdentity checks the built-in Prometheus identity's effective permissions for both certificates and tokens.
func GuardMetricsIdentity(extractor authn.IdentityExtractor) authn.IdentityExtractor {
	return metricsExtractor{extractor}
}

func (e metricsExtractor) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, *authn.ExtractorError) {
	id, err := e.IdentityExtractor.IdentityForRequest(ctx, ri)
	if id == nil || err != nil {
		return id, err
	}
	provider := id.ExternalAuthProvider()
	if provider == nil || provider.ID() != authProviderID ||
		!slices.Contains(id.Attributes()[authproviders.NameAttribute], env.SecureMetricsClientCertCN.Setting()) {
		return id, nil
	}
	if err := permissionsUtils.ValidateScopedReadOnlyPermissions(id.Permissions()); err != nil {
		return nil, authn.NewExtractorError("userpki", "OpenShift Prometheus permissions must be scoped and read-only", err)
	}
	return id, nil
}
