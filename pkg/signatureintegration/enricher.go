package signatureintegration

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var log = logging.LoggerForModule()

// Getter provides access to signature integration data.
type Getter interface {
	GetSignatureIntegration(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error)
}

// integrationReadContext creates a SAC context with Integration read access.
func integrationReadContext(ctx context.Context) context.Context {
	return sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.ResourceScopeKeys(resources.Integration),
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		),
	)
}

// EnrichVerificationResults populates the VerifierName field in all
// signature verification results.
func EnrichVerificationResults(ctx context.Context, getter Getter, results []*storage.ImageSignatureVerificationResult) {
	integrationCtx := integrationReadContext(ctx)
	for _, result := range results {
		name, err := getVerifierNameWithCtx(integrationCtx, getter, result.GetVerifierId())
		if err != nil {
			log.Debugf("Failed to get signature integration name for ID %s: %v", result.GetVerifierId(), err)
			continue
		}
		result.VerifierName = name
	}
}

// GetVerifierName returns the verifier name for a single verification result.
// This is useful for lazy lookups in GraphQL resolvers.
func GetVerifierName(ctx context.Context, getter Getter, verifierID string) (string, error) {
	integrationCtx := integrationReadContext(ctx)
	return getVerifierNameWithCtx(integrationCtx, getter, verifierID)
}

// getVerifierNameWithCtx is an internal helper that assumes the context already
// has the necessary SAC permissions. This avoids re-wrapping the context when
// enriching multiple results.
func getVerifierNameWithCtx(ctx context.Context, getter Getter, verifierID string) (string, error) {
	if verifierID == "" {
		return "", nil
	}

	integration, found, err := getter.GetSignatureIntegration(ctx, verifierID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", nil
	}
	return integration.GetName(), nil
}
