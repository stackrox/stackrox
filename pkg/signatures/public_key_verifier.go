package signatures

import "github.com/stackrox/rox/generated/storage"

type publicKeyVerifier struct {
	publicKeysBase64Enc []string
}

func newPublicKeyVerifier(config *storage.SignatureVerificationConfig_PublicKey) *publicKeyVerifier {
	return &publicKeyVerifier{publicKeysBase64Enc: config.PublicKey.GetPublicKeysBase64Enc()}
}

// VerifySignature implements the SignatureVerifier interface.
// TODO: Right now only a stub implementation for the first abstraction.
func (c *publicKeyVerifier) VerifySignature(rawSignature []byte) (storage.ImageSignatureVerificationResult_Status, error) {
	return storage.ImageSignatureVerificationResult_UNSET, nil
}
