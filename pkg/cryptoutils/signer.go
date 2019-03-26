package cryptoutils

import (
	"io"
)

// Signer provides a simplified interface to signing arbitrary data and verifying the resulting signatures.
type Signer interface {
	SignatureVerifier

	// Sign signs the given data, reading entropy from the given entropy source. The signature is returned as raw bytes.
	Sign(data []byte, entropySrc io.Reader) (sig []byte, err error)
}
