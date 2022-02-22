package signatures

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
)

var (
	log = logging.LoggerForModule()
)

// SignatureVerifier is responsible for verifying signatures using a specific signature verification method.
type SignatureVerifier interface {
	// VerifySignature will take a raw signature and verify it using a specific verification method.
	// It will return a storage.ImageSignatureVerificationResult_Status and
	// an error if the verification was unsuccessful.
	VerifySignature(ctx context.Context, image *storage.Image) (storage.ImageSignatureVerificationResult_Status, error)
}

// NewSignatureVerifier creates a new signature verifier capable of verifying signatures against the provided config.
func NewSignatureVerifier(config *storage.SignatureVerificationConfig) (SignatureVerifier, error) {
	switch cfg := config.GetConfig().(type) {
	case *storage.SignatureVerificationConfig_CosignVerification:
		return newCosignPublicKeyVerifier(cfg.CosignVerification)
	default:
		// Should theoretically never happen.
		return nil, errox.Newf(errox.InvariantViolation,
			"invalid type for signature verification config: %T", cfg)
	}
}

// VerifyAgainstSignatureIntegration is a wrapper that will verify an image signature with SignatureVerifier's created
// based off of the configuration within the storage.SignatureIntegration.
// NOTE: No error will be returned if the SignatureVerifier's creation failed or the signature verification itself
// failed. A log entry will be created for a failing creation, and the verification status can be must be checked within
// the storage.ImageSignatureVerificationResult.
func VerifyAgainstSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration,
	image *storage.Image) []storage.ImageSignatureVerificationResult {
	verifiers := make([]SignatureVerifier, 0, len(integration.GetSignatureVerificationConfigs()))
	for _, config := range integration.GetSignatureVerificationConfigs() {
		verifier, err := NewSignatureVerifier(config)
		if err != nil {
			log.Errorf("Error during creation of the signature verifier for config %q: %v",
				config.GetId(), err)
			continue
		}
		verifiers = append(verifiers, verifier)
	}

	var results []storage.ImageSignatureVerificationResult
	for _, verifier := range verifiers {
		res, err := verifier.VerifySignature(ctx, image)
		// We do not currently support specifying which specific method within an image signature integration should
		// be successful. Hence, short-circuit on the first successfully verified signature within an image signature
		// integration.
		if res == storage.ImageSignatureVerificationResult_VERIFIED {
			return []storage.ImageSignatureVerificationResult{
				{
					VerificationTime: protoconv.ConvertTimeToTimestamp(time.Now()),
					VerifierId:       integration.GetId(),
					Status:           res,
				},
			}
		}
		// Right now, we will duplicate the verification result for each SignatureVerifier contained within an image
		// signature, ensuring all errors are properly returned to the caller.
		verificationResult := storage.ImageSignatureVerificationResult{
			VerificationTime: protoconv.ConvertTimeToTimestamp(time.Now()),
			VerifierId:       integration.GetId(),
			Status:           res,
		}

		if err != nil {
			verificationResult.Description = err.Error()
		}

		results = append(results, verificationResult)
	}
	return results
}

// VerifyAgainstSignatureIntegrations is a wrapper that will verify an image signature against a list of
// storage.SignatureIntegration using VerifyAgainstSignatureIntegration.
// NOTE: No error will be returned if the SignatureVerifier's creation failed or the signature verification itself
// failed. A log entry will be created for a failing creation, and the verification status can be must be checked within
// the storage.ImageSignatureVerificationResult.
func VerifyAgainstSignatureIntegrations(ctx context.Context, integrations []*storage.SignatureIntegration,
	image *storage.Image) (map[*storage.SignatureIntegration][]storage.ImageSignatureVerificationResult, error) {
	results := make(map[*storage.SignatureIntegration][]storage.ImageSignatureVerificationResult, len(integrations))
	var verifierCreationErrs error
	for _, integration := range integrations {
		verificationResults := VerifyAgainstSignatureIntegration(ctx, integration, image)
		results[integration] = verificationResults
	}
	return results, verifierCreationErrs
}
