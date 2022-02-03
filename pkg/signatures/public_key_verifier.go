package signatures

import (
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

const publicKeyType = "PUBLIC KEY"

type publicKeyVerifier struct {
	parsedPublicKeys []crypto.PublicKey
}

// newPublicKeyVerifier creates a public key verifier with the given configuration.
// It will return an error if the provided public keys could not be parsed or the base64 decoding failed.
func newPublicKeyVerifier(config *storage.SignatureVerificationConfig_PublicKey) (*publicKeyVerifier, error) {
	base64EncPublicKeys := config.PublicKey.GetPublicKeysBase64Enc()

	parsedKeys := make([]crypto.PublicKey, 0, len(base64EncPublicKeys))
	for _, base64EncKey := range base64EncPublicKeys {
		// Each key should be base64 encoded.
		decodedKey, err := base64.StdEncoding.DecodeString(base64EncKey)
		if err != nil {
			return nil, errors.Wrap(err, "decoding base64 encoded key")
		}

		// We expect the key to be PEM encoded. There should be no rest returned after decoding.
		keyBlock, rest := pem.Decode(decodedKey)
		if keyBlock == nil || keyBlock.Type != publicKeyType || rest != nil {
			return nil, errorhelpers.NewErrInvariantViolation(
				"failed to decode PEM block containing public key")
		}

		parsedKey, err := x509.ParsePKIXPublicKey(keyBlock.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "parsing DER encoded public key")
		}
		parsedKeys = append(parsedKeys, parsedKey)
	}

	return &publicKeyVerifier{parsedPublicKeys: parsedKeys}, nil
}

// VerifySignature implements the SignatureVerifier interface.
// TODO: Right now only a stub implementation for the first abstraction.
func (c *publicKeyVerifier) VerifySignature(rawSignature []byte) (storage.ImageSignatureVerificationResult_Status, error) {
	return storage.ImageSignatureVerificationResult_UNSET, nil
}
