//go:build !amd64 || !goexperiment.simd

package simdutil

import (
	"encoding/binary"
)

// Global constants for private network ranges
// These are reused across all function calls to avoid repeated array initialization
var (
	// IPv4 private network masks and prefixes
	// Ordered by likelihood (10.x.x.x and 192.168.x.x are most common)
	ipv4Masks = [5]uint32{
		0xFF000000, // 10.0.0.0/8
		0xFFFF0000, // 192.168.0.0/16
		0xFFF00000, // 172.16.0.0/12
		0xFFC00000, // 100.64.0.0/10
		0xFFFF0000, // 169.254.0.0/16
	}
	ipv4Prefixes = [5]uint32{
		0x0A000000, // 10.0.0.0
		0xC0A80000, // 192.168.0.0
		0xAC100000, // 172.16.0.0
		0x64400000, // 100.64.0.0
		0xA9FE0000, // 169.254.0.0
	}
)

// CheckIPv4Public determines if an IPv4 address is public (not in private ranges).
// This is the fallback implementation for non-AMD64 platforms or when SIMD is disabled.
func CheckIPv4Public(d [4]byte) bool {
	// Convert IP to uint32 in network byte order (big-endian)
	ip := binary.BigEndian.Uint32(d[:])

	// Check each private network range using global constants
	for i := 0; i < 5; i++ {
		if (ip & ipv4Masks[i]) == ipv4Prefixes[i] {
			return false // Is private
		}
	}

	return true // Is public
}
