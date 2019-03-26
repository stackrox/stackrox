package cryptoutils

// SignatureVerifier abstracts the functionality of verifying cryptographic signatures.
type SignatureVerifier interface {
	// Verify verifies that the given signature matches the data. If the signature matches, nil is returned, otherwise
	// an error is returned.
	Verify(data, sig []byte) (err error)
}
