package certgen

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

const (
	// JWTKeyPEMFileName is the canonical file name (basename) of the file storing the key for signing JWTs,
	// encoded in PEM format (default).
	JWTKeyPEMFileName = "jwt-key.pem"
	// JWTKeyDERFileName is the canonical file name (basename) of the file storing the key for signing JWTs,
	// encoded in DER format (legacy).
	JWTKeyDERFileName = "jwt-key.der"
)

// GenerateJWTSigningKey generates a new RSA private key that can be used to sign JWTs.
func GenerateJWTSigningKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 4096)
}

// AddJWTSigningKeyToFileMap adds the PEM-encoded JWT signing key to the given file map.
func AddJWTSigningKeyToFileMap(fileMap map[string][]byte, jwtKey *rsa.PrivateKey) {
	fileMap[JWTKeyPEMFileName] = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(jwtKey),
	})
}

// LoadJWTSigningKeyFromFileMap loads the JWT signing key from the given file map. Both
// PEM and DER-encoded keys are supported.
func LoadJWTSigningKeyFromFileMap(fileMap map[string][]byte) (*rsa.PrivateKey, error) {
	keyPEM := fileMap[JWTKeyPEMFileName]
	if len(keyPEM) == 0 {
		keyDER := fileMap[JWTKeyDERFileName]
		if len(keyDER) == 0 {
			return nil, errors.New("file map contains neither PEM nor DER-encoded JWT signing key")
		}
		return x509.ParsePKCS1PrivateKey(keyDER)
	}
	return jwt.ParseRSAPrivateKeyFromPEM(keyPEM)
}
