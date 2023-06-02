package signatures

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
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
func NewSignatureVerifier(config *storage.CosignPublicKeyVerification) (SignatureVerifier, error) {
	return newCosignPublicKeyVerifier(config)
}

// NewSignatureFetcher creates a new signature fetcher capable of fetching a specific signature format for an image.
// Currently, only cosign public key signatures are supported.
func NewSignatureFetcher() SignatureFetcher {
	return newCosignPublicKeySignatureFetcher()
}

// VerifyAgainstSignatureIntegration is a wrapper that will verify an image signature with SignatureVerifier's created
// based off of the configuration within the storage.SignatureIntegration.
// NOTE: No error will be returned if the SignatureVerifier's creation failed or the signature verification itself
// failed. A log entry will be created for a failing creation, and the verification status can be must be checked within
// the storage.ImageSignatureVerificationResult.
func VerifyAgainstSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration,
	image *storage.Image) []*storage.ImageSignatureVerificationResult {
	verifiers := createVerifiersFromIntegration(integration)
	var results []*storage.ImageSignatureVerificationResult
	for _, verifier := range verifiers {
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
			return []*storage.ImageSignatureVerificationResult{
				verificationResult,
			}
		}
		// Right now, we will duplicate the verification result for each SignatureVerifier contained within an image
		// signature, ensuring all errors are properly returned to the caller.
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
	image *storage.Image) []*storage.ImageSignatureVerificationResult {
	// If signature fetching is disabled, it also doesn't make much sense to verify signatures, hence skip it.
	if env.DisableSignatureFetching.BooleanSetting() {
		return nil
	}

	var results []*storage.ImageSignatureVerificationResult
	for _, integration := range integrations {
		verificationResults := VerifyAgainstSignatureIntegration(ctx, integration, image)
		results = append(results, verificationResults...)
	}
	return results
}

func createVerifiersFromIntegration(integration *storage.SignatureIntegration) []SignatureVerifier {
	verifiers := make([]SignatureVerifier, 0)

	// This method will later be extended with other verification methods.
	if integration.GetCosign() != nil {
		verifier, err := NewSignatureVerifier(integration.GetCosign())
		if err != nil {
			log.Errorf("Error during creation of the signature verifier for signature integration %q: %v",
				integration.GetId(), err)
		} else {
			verifiers = append(verifiers, verifier)
		}
	}

	return verifiers
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
		fetchedSignatures, err = fetchAndAppendSignatures(ctx, fetcher, image, fullImageName, registry, fetchedSignatures)
		return err
	},
		retry.Tries(2),
		retry.OnlyRetryableErrors(),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(500 * time.Millisecond)
		}))

	return fetchedSignatures, err
}

func fetchAndAppendSignatures(ctx context.Context, fetcher SignatureFetcher, image *storage.Image,
	fullImageName string, registry registryTypes.Registry, fetchedSignatures []*storage.Signature) ([]*storage.Signature, error) {
	sigFetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sigs, err := fetcher.FetchSignatures(sigFetchCtx, image, fullImageName, registry)
	if err != nil {
		return nil, err
	}

	for _, sig := range sigs {
		// TODO(ROX-9688): Replace with generated generic contains function.
		if !containsSignature(sig, fetchedSignatures) {
			fetchedSignatures = append(fetchedSignatures, sig)
		}
	}
	return fetchedSignatures, nil
}

func containsSignature(sig *storage.Signature, sigs []*storage.Signature) bool {
	for _, s := range sigs {
		if proto.Equal(sig, s) {
			return true
		}
	}
	return false
}
