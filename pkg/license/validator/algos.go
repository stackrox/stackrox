package validator

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"

	// Ensure SHA256 and SHA384 hash functions are available.
	_ "crypto/sha256"
	_ "crypto/sha512"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/cryptoutils"
)

type signatureVerifierCreator func(publicKeyData []byte) (cryptoutils.SignatureVerifier, error)

const (
	// EC256 uses the ECDSA algorithm with a P-256 curve and the SHA-256 digest algorithm.
	EC256 = "ec256-sha256"
	// EC384 uses the ECDSA algorithm with a P-384 curve and the SHA-384 digest algorithm.
	EC384 = "ec384-sha384"
)

var (
	signatureVerifierByName = map[string]signatureVerifierCreator{
		EC256: createECDSAVerifierCreator(crypto.SHA256),
		EC384: createECDSAVerifierCreator(crypto.SHA384),
	}
)

func createECDSAVerifierCreator(digestAlgo crypto.Hash) signatureVerifierCreator {
	return func(publicKeyData []byte) (cryptoutils.SignatureVerifier, error) {
		publicKey, err := x509.ParsePKIXPublicKey(publicKeyData)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse PKIX public key data")
		}

		if ecdsaKey, ok := publicKey.(*ecdsa.PublicKey); ok {
			return cryptoutils.NewECDSAVerifier(ecdsaKey, digestAlgo), nil
		}
		return nil, fmt.Errorf("parsed public key is not an ECDSA key but is: %T", publicKey)
	}
}
