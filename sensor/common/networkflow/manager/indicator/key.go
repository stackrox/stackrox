package indicator

import (
	"hash"
	"unsafe"

	"github.com/cespare/xxhash"
	"github.com/stackrox/rox/generated/storage"
)

var hashDelimiter = []byte{0}

// keyHash produces a string that uniquely identifies a given NetworConn indicator.
// Assumption: Two NetworkConn's are identical (for the network-graph purposes) when their keys are identical.
// This is memory-optimized implementation that is slower than `keyString`, but the resulting string takes less memory.
func (i *NetworkConn) keyHash() string {
	h := xxhash.New()
	// Collision probability example: for 100M uniformly distributed items, the collision probability is 2.71x10^4 = 0.027.
	// For lower collision probabilities, one needs to use a fast 128bit hash, for example: XXH3_128 (LLM recommendation).
	hashStrings(h, i.SrcEntity.ID, i.DstEntity.ID)
	hashPortAndProtocol(h, i.DstPort, i.Protocol)
	return hashToHexString(h.Sum64())
}

// Common hash computation utilities
func hashPortAndProtocol(h hash.Hash64, port uint16, protocol storage.L4Protocol) {
	buf := [6]byte{
		byte(port >> 8), byte(port),
		byte(protocol >> 24), byte(protocol >> 16),
		byte(protocol >> 8), byte(protocol),
	}
	_, _ = h.Write(buf[:]) // xxhash never returns errors, but being explicit
}

// hashToHexString is performance-optimized implementation of fmt.Sprintf("%016x", hash).
// Benchmark summary:
// Speed: Current implementation is 4x faster (15.14ns vs 61.87ns) than fmt.Sprintf.
// Memory: Current uses less memory (16B vs 24B) and a single allocation (1 vs 2).
// Resulting string is identical in both cases.
func hashToHexString(hash uint64) string {
	const hexDigits = "0123456789abcdef"
	buf := make([]byte, 16)
	// Process 4 bits at a time from right to left
	for i := 15; i >= 0; i-- {
		buf[i] = hexDigits[hash&0xF]
		hash >>= 4
	}

	return string(buf)
}

func hashStrings(h hash.Hash64, strs ...string) {
	for i, s := range strs {
		if i > 0 {
			_, _ = h.Write(hashDelimiter) // Use null byte as delimiter to avoid hash collisions
		}
		// Use zero-copy conversion from string to []byte using unsafe to avoid allocation.
		// This is safe because:
		// 1. h.Write() doesn't modify data (io.Writer contract)
		// 2. xxhash doesn't retain references
		// 3. string s remains alive during the call
		if len(s) > 0 {
			//#nosec G103 -- Audited: zero-copy string-to-bytes conversion for performance
			b := unsafe.Slice(unsafe.StringData(s), len(s))
			_, _ = h.Write(b)
		}
	}
}

// Binary key generation methods for ContainerEndpoint

// binaryKeyHash produces a binary hash that uniquely identifies a given ContainerEndpoint indicator.
// This is a memory-optimized implementation using direct hash generation without string conversion.
func (i *ContainerEndpoint) binaryKeyHash() BinaryHash {
	h := xxhash.New()
	hashStrings(h, i.Entity.ID)
	hashPortAndProtocol(h, i.Port, i.Protocol)
	return BinaryHash(h.Sum64())
}

// Binary key generation methods for ProcessListening

// binaryKeyHash produces a binary hash that uniquely identifies a given ProcessListening indicator.
// This is a memory-optimized implementation using direct hash generation without string conversion.
func (i *ProcessListening) binaryKeyHash() BinaryHash {
	h := xxhash.New()
	// From `ProcessIndicatorUniqueKey` - identifies the process and the container
	hashStrings(h, i.PodID, i.ContainerName, i.Process.ProcessName, i.Process.ProcessExec, i.Process.ProcessArgs)
	// From: containerEndpoint - identifies the endpoint
	hashPortAndProtocol(h, i.Port, i.Protocol)
	return BinaryHash(h.Sum64())
}
