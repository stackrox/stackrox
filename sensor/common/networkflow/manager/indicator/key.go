package indicator

import (
	"encoding/hex"
	"hash"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// Key produces a string that uniquely identifies a given NetworConn indicator.
// Assumption: Two NetworkConn's are identical (for the network-graph purposes) when their keys are identical.
// This is a CPU-optimized implementation that is faster than `keyHash`, but the resulting string takes more memory.
func (i *NetworkConn) keyString() string {
	var buf strings.Builder
	// 82 chars is an estimate based on typical string-lengths of the NetworkConn's fields to avoid re-sizing.
	// 3 chars of the delimiters can be saved, but would only reduce number of bytes allocated locally and
	// won't reduce the size of a large collection holding many NetworkConn's.
	buf.Grow(82)
	buildStringKey(&buf, i.SrcEntity.ID, i.DstEntity.ID) // 2 x 36 chars for UUIDv4 + 1 char for delimiter
	formatPortAndProtocol(&buf, i.DstPort, i.Protocol)   // 9 chars maximally
	return buf.String()
}

// keyHash produces a string that uniquely identifies a given NetworConn indicator.
// Assumption: Two NetworkConn's are identical (for the network-graph purposes) when their keys are identical.
// This is memory-optimized implementation that is slower than `keyString`, but the resulting string takes less memory.
func (i *NetworkConn) keyHash() string {
	h := fnv.New64a()
	// For a collection of length 10^N, the 64bit FNV-1a hash has approximate collision probability of 2.71x10^(N-4).
	// For example: for 100M uniformly distributed items, the collision probability is 2.71x10^4 = 0.027.
	// For lower collision probabilities, one needs to use a fast 128bit hash, for example: XXH3_128 (LLM recommendation).
	hashStrings(h, i.SrcEntity.ID, i.DstEntity.ID)
	hashPortAndProtocol(h, i.DstPort, i.Protocol)
	return hashToHexString(h.Sum64())
}

// keyString produces a string that uniquely identifies a given ContainerEndpoint indicator.
// Assumption: Two ContainerEndpoint's are identical (for the network-graph purposes) when their keys are identical.
// This is a CPU-optimized implementation that is faster than `keyHash`, but the resulting string takes more memory.
func (i *ContainerEndpoint) keyString() string {
	var buf strings.Builder
	buf.Grow(45)                                    // Estimate based on typical ID lengths.
	_, _ = buf.WriteString(i.Entity.ID)             // 36 chars (UUIDv4)
	formatPortAndProtocol(&buf, i.Port, i.Protocol) // 9 chars maximally

	return buf.String()
}

// keyHash produces a string that uniquely identifies a given ContainerEndpoint indicator.
// Assumption: Two ContainerEndpoint's are identical (for the network-graph purposes) when their keys are identical.
// This is memory-optimized implementation that is slower than `keyString`, but the resulting string takes less memory.
func (i *ContainerEndpoint) keyHash() string {
	h := fnv.New64a()
	hashStrings(h, i.Entity.ID)
	hashPortAndProtocol(h, i.Port, i.Protocol)
	return hashToHexString(h.Sum64())
}

// keyString produces a string that uniquely identifies a given ProcessListening indicator.
// Assumption: Two ProcessListening's are identical (for the network-graph & PLoP purposes) when their keys are identical.
// This is a CPU-optimized implementation that is faster than `keyHash`, but the resulting string takes more memory.
func (i *ProcessListening) keyString() string {
	var buf strings.Builder
	// It is hard to compute any reasonable size for pre-allocation as many items have variable length.
	// Estimating partially based on gut feeling.
	buf.Grow(165)

	// Skipping some fields to save memory - they should not be required to ensure uniqueness.
	// 5 x strings with variable length (assuming 30 chars each) + 5 chars for delimiter = 155 chars
	buildStringKey(&buf, i.PodID, i.ContainerName, i.Process.ProcessName, i.Process.ProcessExec, i.Process.ProcessArgs)
	formatPortAndProtocol(&buf, i.Port, i.Protocol) // 9 chars maximally
	return buf.String()
}

// keyHash produces a string that uniquely identifies a given ProcessListening indicator.
// Assumption: Two ProcessListening's are identical (for the network-graph & PLoP purposes) when their keys are identical.
// This is memory-optimized implementation that is slower than `keyString`, but the resulting string takes less memory.
func (i *ProcessListening) keyHash() string {
	h := fnv.New64a()
	// From `ProcessIndicatorUniqueKey` - identifies the process and the container
	hashStrings(h, i.PodID, i.ContainerName, i.Process.ProcessName, i.Process.ProcessExec, i.Process.ProcessArgs)
	// From: containerEndpoint - identifies the endpoint
	hashPortAndProtocol(h, i.Port, i.Protocol)
	return hashToHexString(h.Sum64())
}

// Common hash computation utilities
func hashPortAndProtocol(h hash.Hash64, port uint16, protocol storage.L4Protocol) {
	portBytes := [2]byte{byte(port >> 8), byte(port)}
	_, _ = h.Write(portBytes[:]) // FNV never returns errors, but being explicit

	protocolBytes := [4]byte{
		byte(protocol >> 24), byte(protocol >> 16),
		byte(protocol >> 8), byte(protocol),
	}
	_, _ = h.Write(protocolBytes[:])
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

func formatPortAndProtocol(buf *strings.Builder, port uint16, protocol storage.L4Protocol) {
	buf.WriteByte(':')
	buf.WriteString(strconv.FormatUint(uint64(port), 10))
	buf.WriteByte(':')
	buf.WriteString(strconv.FormatUint(uint64(protocol), 10))
}

func buildStringKey(buf *strings.Builder, parts ...string) {
	for i, part := range parts {
		if i > 0 {
			buf.WriteByte(':')
		}
		buf.WriteString(part)
	}
}

func hashStrings(h hash.Hash64, strs ...string) {
	for i, s := range strs {
		if i > 0 {
			_, _ = h.Write([]byte{0}) // Use null byte as delimiter to avoid hash collisions
		}
		_, _ = h.Write([]byte(s))
	}
}
