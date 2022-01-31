package signatures

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// SignatureVerifier is responsible for verifying signatures using a specific signature verification method.
type SignatureVerifier interface {
	// VerifySignature will take a raw signature and verify it using a specific verification method.
	// It will return a storage.ImageSignatureVerificationResult_Status and an error if the verification was unsuccessful.
	VerifySignature(rawSignature []byte) (storage.ImageSignatureVerificationResult_Status, error)
}

func NewSignatureVerifier(config *storage.SignatureVerificationConfig) (SignatureVerifier, error) {
	switch cfg := config.GetConfig().(type) {
	case *storage.SignatureVerificationConfig_PublicKey:
		return newPublicKeyVerifier(cfg), nil
	default:
		// Should theoretically never happen.
		return nil, errorhelpers.NewErrInvariantViolation(fmt.Sprintf(
			"invalid type for signature verification config: %t", cfg))
	}
}
