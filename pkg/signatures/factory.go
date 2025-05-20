package signatures

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
)

var (
	log = logging.LoggerForModule()
)

// SignatureVerifier is responsible for verifying signatures using a specific signature verification method.
type SignatureVerifier interface {
	// VerifySignature will take an image and verify its signature using a specific verification method.
	// It will return a storage.ImageSignatureVerificationResult_Status, the verified image references if verification
	// was successful, and an error if the verification was unsuccessful.
	VerifySignature(ctx context.Context, image *storage.Image) (storage.ImageSignatureVerificationResult_Status, []string, error)
}

// SignatureFetcher is responsible for fetching raw signatures supporting multiple specific signature formats.
type SignatureFetcher interface {
	FetchSignatures(ctx context.Context, image *storage.Image, fullImageName string, registry registryTypes.Registry) ([]*storage.Signature, error)
}

// NewSignatureVerifier creates a new signature verifier capable of verifying signatures against the provided config.
func NewSignatureVerifier(config *storage.SignatureIntegration) (SignatureVerifier, error) {
	return newCosignSignatureVerifier(config)
}

// NewSignatureFetcher creates a new signature fetcher capable of fetching a specific signature format for an image.
// Currently, only cosign public key signatures are supported.
func NewSignatureFetcher() SignatureFetcher {
	return newCosignSignatureFetcher()
}

// VerifyAgainstSignatureIntegration is a wrapper that will verify an image signature with SignatureVerifier's created
// based off of the configuration within the storage.SignatureIntegration.
// NOTE: No error will be returned if the SignatureVerifier's creation failed or the signature verification itself
// failed, the status can be checked in the verification result.
func VerifyAgainstSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration,
	image *storage.Image) *storage.ImageSignatureVerificationResult {
	verifier, err := createVerifierFromSignatureIntegration(integration)
	if err != nil {
		return &storage.ImageSignatureVerificationResult{
			VerificationTime: protoconv.ConvertTimeToTimestamp(time.Now()),
			VerifierId:       integration.GetId(),
			Status:           storage.ImageSignatureVerificationResult_GENERIC_ERROR,
			Description:      err.Error(),
		}
	}
	res, verifiedImageReferences, err := verifier.VerifySignature(ctx, image)

	verificationResult := &storage.ImageSignatureVerificationResult{
		VerificationTime:        protoconv.ConvertTimeToTimestamp(time.Now()),
		VerifierId:              integration.GetId(),
		Status:                  res,
		VerifiedImageReferences: verifiedImageReferences,
	}
	// We do not currently support specifying which specific method within an image signature integration should
	// be successful. Hence, short-circuit on the first successfully verified signature within an image signature
	// integration.
	if res == storage.ImageSignatureVerificationResult_VERIFIED {
		return verificationResult
	}
	// Right now, we will duplicate the verification result for each SignatureVerifier contained within an image
	// signature, ensuring all errors are properly returned to the caller.
	if err != nil {
		verificationResult.Description = err.Error()
	}
	return verificationResult
}

// VerifyAgainstSignatureIntegrations is a wrapper that will verify an image signature against a list of
// storage.SignatureIntegration using VerifyAgainstSignatureIntegration.
// NOTE: No error will be returned if the SignatureVerifier's creation failed or the signature verification itself
// failed. A log entry will be created for a failing creation, and the verification status can be must be checked within
// the storage.ImageSignatureVerificationResult.
func VerifyAgainstSignatureIntegrations(ctx context.Context, integrations []*storage.SignatureIntegration,
	image *storage.Image) []*storage.ImageSignatureVerificationResult {
	// If signature fetching is disabled, it also doesn't make much sense to verify signatures, hence skip it.
	if env.DisableSignatureFetching.BooleanSetting() {
		return nil
	}

	var results []*storage.ImageSignatureVerificationResult
	for _, integration := range integrations {
		verificationResults := VerifyAgainstSignatureIntegration(ctx, integration, image)
		results = append(results, verificationResults)
	}
	return results
}

func createVerifierFromSignatureIntegration(integration *storage.SignatureIntegration) (SignatureVerifier, error) {
	verifier, err := NewSignatureVerifier(integration)
	if err != nil {
		log.Errorf("Error during creation of the signature verifier for signature integration %q: %v",
			integration.GetId(), err)
		return nil, err
	}
	return verifier, nil
}

// FetchImageSignaturesWithRetries will try and fetch signatures for the given image from the given registry and return them.
// It will retry on transient errors and return the fetched signatures.
func FetchImageSignaturesWithRetries(ctx context.Context, fetcher SignatureFetcher, image *storage.Image,
	fullImageName string, registry registryTypes.Registry) ([]*storage.Signature, error) {
	// Short-circuit if signature fetching is disabled.
	if env.DisableSignatureFetching.BooleanSetting() {
		return nil, nil
	}

	var fetchedSignatures []*storage.Signature
	var err error
	err = retry.WithRetry(func() error {
		sigFetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		fetchedSignatures, err = fetcher.FetchSignatures(sigFetchCtx, image, fullImageName, registry)
		return err
	},
		retry.WithContext(ctx),
		retry.Tries(5),
		retry.OnlyRetryableErrors(),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(500 * time.Millisecond)
		}))

	return fetchedSignatures, err
}
