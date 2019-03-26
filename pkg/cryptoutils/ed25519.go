package cryptoutils

import (
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/ed25519"
)

// NewED25519Verifier returns a verifier for ED25519 signatures.
func NewED25519Verifier(publicKey ed25519.PublicKey) SignatureVerifier {
	return &ed25519Verifier{
		publicKey: publicKey,
	}
}

type ed25519Verifier struct {
	publicKey ed25519.PublicKey
}

func (v *ed25519Verifier) Verify(data, sig []byte) error {
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length %d", len(sig))
	}
	if !ed25519.Verify(v.publicKey, data, sig) {
		return errors.New("signature verification failed")
	}
	return nil
}

// NewED25519Signer returns a new signer using the ED25519 algorithm and the given private key.
func NewED25519Signer(pk ed25519.PrivateKey) Signer {
	return &ed25519Signer{
		ed25519Verifier: ed25519Verifier{pk.Public().(ed25519.PublicKey)},
		priv:            pk,
	}
}

type ed25519Signer struct {
	ed25519Verifier
	priv ed25519.PrivateKey
}

func (s *ed25519Signer) Sign(data []byte, entropySrc io.Reader) ([]byte, error) {
	return ed25519.Sign(s.priv, data), nil
}
