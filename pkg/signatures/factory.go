package signatures

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
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
