package confighash

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeCAHash computes the SHA256 hash of the given CA PEM and returns the hex encoded string.
func ComputeCAHash(caPEM []byte) string {
	sum := sha256.Sum256(caPEM)
	return hex.EncodeToString(sum[:])
}
