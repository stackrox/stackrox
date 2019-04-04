package license

import (
	"crypto/sha1"
	"encoding/hex"
)

// SigningKeyFingerprint returns the fingerprint for the given public key raw bytes.
// The fingerprint is the SHA1 fingerprint of the DER-encoded key. From a public key PEM
// file, the fingerprint can be generated via
// openssl pkey -pubin -in pubkey.pem -inform PEM -outform DER | openssl sha1
func SigningKeyFingerprint(pubKeyBytes []byte) string {
	hash := sha1.New()
	_, _ = hash.Write(pubKeyBytes)

	return hex.EncodeToString(hash.Sum(nil))
}
