package signatures

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoconv"
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
// It will return an error if the creation of SignatureVerifier's fails or the verification of the signature fails.
func VerifyAgainstSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration, image *storage.Image) ([]storage.ImageSignatureVerificationResult, error) {
	verifiers := make([]SignatureVerifier, 0, len(integration.GetSignatureVerificationConfigs()))
	for _, config := range integration.GetSignatureVerificationConfigs() {
		verifier, err := NewSignatureVerifier(config)
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, verifier)
	}

	var results []storage.ImageSignatureVerificationResult
	var allVerifyErrs error
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
			}, nil
		}
		// Right now, we will duplicate the verification result for each SignatureVerifier contained within an image
		// signature, ensuring all errors are properly returned to the caller.
		results = append(results, storage.ImageSignatureVerificationResult{
			VerificationTime: protoconv.ConvertTimeToTimestamp(time.Now()),
			VerifierId:       integration.GetId(),
			Status:           res,
			Description:      err.Error(),
		})
		allVerifyErrs = multierror.Append(allVerifyErrs, err)
	}
	return results, allVerifyErrs
}

// VerifyAgainstSignatureIntegrations is a wrapper that will verify an image signature against a list of
// storage.SignatureIntegration using VerifyAgainstSignatureIntegration.
// It will return a map of integrations and their related results and any error that may have occurred.
func VerifyAgainstSignatureIntegrations(ctx context.Context, integrations []*storage.SignatureIntegration,
	image *storage.Image) (map[*storage.SignatureIntegration][]storage.ImageSignatureVerificationResult, error) {
	results := make(map[*storage.SignatureIntegration][]storage.ImageSignatureVerificationResult, len(integrations))

	var allVerifyErrs error

	for _, integration := range integrations {
		verificationResults, err := VerifyAgainstSignatureIntegration(ctx, integration, image)
		results[integration] = verificationResults
		if err != nil {
			allVerifyErrs = multierror.Append(allVerifyErrs, err)
		}
	}
	return results, allVerifyErrs
}
