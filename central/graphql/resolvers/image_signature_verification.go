package resolvers

import (
	"context"

	"github.com/stackrox/rox/pkg/signatureintegration"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolver("ImageSignatureVerificationResult", "verifierName: String!"),
	)
}

// VerifierName returns the signature integration name. If the name was
// pre-populated by the service, it is returned directly. Otherwise, a lookup
// is performed from the SignatureIntegrationDataStore.
func (resolver *imageSignatureVerificationResultResolver) VerifierName(ctx context.Context) (string, error) {
	// Short-circuit if the name was already populated by the service.
	if name := resolver.data.GetVerifierName(); name != "" {
		return name, nil
	}
	return signatureintegration.GetVerifierName(
		ctx,
		resolver.root.SignatureIntegrationDataStore,
		resolver.data.GetVerifierId(),
	)
}
