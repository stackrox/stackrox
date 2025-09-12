package confighash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeCAHash(t *testing.T) {
	input1 := []byte("first input")
	input2 := []byte("second input")

	hash1 := ComputeCAHash(input1)
	hash1_2 := ComputeCAHash(input1)
	hash2 := ComputeCAHash(input2)

	assert.Equal(t, hash1, hash1_2, "Hash function should be deterministic")
	assert.NotEqual(t, hash1, hash2, "Different inputs should produce different hashes")
	assert.Len(t, hash1, 64, "SHA256 hex string should be 64 characters long")
	assert.Len(t, hash2, 64, "SHA256 hex string should be 64 characters long")
}
