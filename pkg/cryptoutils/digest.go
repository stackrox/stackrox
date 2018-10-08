package cryptoutils

import "crypto"

// ComputeDigest returns the digest of the given data computed by the specified cryptographic hash algorithm.
func ComputeDigest(data []byte, hash crypto.Hash) ([]byte, error) {
	hasher := hash.New()
	if _, err := hasher.Write(data); err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}
