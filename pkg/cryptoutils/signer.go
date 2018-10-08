package cryptoutils

import (
	"crypto"
	"crypto/ecdsa"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"

	"golang.org/x/crypto/ed25519"
)

// Signer provides a simplified interface to signing arbitrary data and verifying the resulting signatures.
type Signer interface {
	// Sign signs the given data, reading entropy from the given entropy source. The signature is returned as raw bytes.
	Sign(data []byte, entropySrc io.Reader) (sig []byte, err error)
	// Verify verifies that the given signature matches the data. If the signature matches, nil is returned, otherwise
	// an error is returned.
	Verify(data, sig []byte) (err error)
}

// NewECDSA256Signer returns a new signer using the ECDSA256 algorithm with the given private key and hash
// function.
func NewECDSA256Signer(pk *ecdsa.PrivateKey, hash crypto.Hash) Signer {
	return &ecdsa256Signer{
		hash: hash,
		priv: pk,
	}
}

type ecdsa256Signer struct {
	hash crypto.Hash
	priv *ecdsa.PrivateKey
}

type ecdsaSig struct {
	R, S *big.Int
}

func (es *ecdsa256Signer) Sign(data []byte, entropySrc io.Reader) ([]byte, error) {
	digest, err := ComputeDigest(data, es.hash)
	if err != nil {
		return nil, fmt.Errorf("computing digest: %v", err)
	}

	r, s, err := ecdsa.Sign(entropySrc, es.priv, digest)
	if err != nil {
		return nil, fmt.Errorf("signing: %v", err)
	}
	sig, err := asn1.Marshal(ecdsaSig{R: r, S: s})
	if err != nil {
		return nil, fmt.Errorf("marshalling: %v", err)
	}
	return sig, nil
}

func (es *ecdsa256Signer) Verify(data, sig []byte) error {
	digest, err := ComputeDigest(data, es.hash)
	if err != nil {
		return fmt.Errorf("computing digest: %v", err)
	}

	var ecdsaSig ecdsaSig
	if rest, err := asn1.Unmarshal(sig, &ecdsaSig); err != nil {
		return fmt.Errorf("unmarshalling signature: %v", err)
	} else if len(rest) != 0 {
		return fmt.Errorf("unmarshalling signature: %d extra bytes", len(rest))
	}
	if !ecdsa.Verify(&es.priv.PublicKey, digest, ecdsaSig.R, ecdsaSig.S) {
		return errors.New("signature verification failed")
	}
	return nil
}

// NewED25519Signer returns a new signer using the ED25519 algorithm and the given private key.
func NewED25519Signer(pk ed25519.PrivateKey) Signer {
	return ed25519Signer{priv: pk}
}

type ed25519Signer struct {
	priv ed25519.PrivateKey
}

func (s ed25519Signer) Sign(data []byte, entropySrc io.Reader) ([]byte, error) {
	return ed25519.Sign(s.priv, data), nil
}

func (s ed25519Signer) Verify(data, sig []byte) error {
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length %d", len(sig))
	}
	if !ed25519.Verify(s.priv.Public().(ed25519.PublicKey), data, sig) {
		return errors.New("signature verification failed")
	}
	return nil
}
