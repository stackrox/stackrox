package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/signatureintegration"
)

// VerifierName looks up the signature integration name at query time.
// This overrides the generated resolver to perform the lookup from the
// SignatureIntegrationDataStore rather than relying on pre-populated data.
func (resolver *imageSignatureVerificationResultResolver) VerifierName(ctx context.Context) (string, error) {
	return signatureintegration.GetVerifierName(
		ctx,
		resolver.root.SignatureIntegrationDataStore,
		resolver.data.GetVerifierId(),
	)
}
