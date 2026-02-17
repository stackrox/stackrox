//go:build amd64 && goexperiment.simd

package net

import (
	"net"
	"testing"

	"github.com/stackrox/rox/pkg/net/internal/simdutil"
	"github.com/stretchr/testify/assert"
)

// TestSIMDCorrectness_PublicIPs verifies that SIMD implementation matches
// scalar implementation for known public IP addresses.
func TestSIMDCorrectness_PublicIPs(t *testing.T) {
	publicIPs := []string{
		"8.8.8.8",              // Google DNS
		"1.1.1.1",              // Cloudflare DNS
		"54.239.28.85",         // AWS
		"151.101.1.140",        // Fastly
		"93.184.216.34",        // Example.com
		"216.58.215.46",        // Google
		"13.107.42.14",         // Microsoft
		"199.232.69.194",       // GitHub
		"2001:4860:4860::8888", // Google DNS IPv6
		"::ffff:8.8.8.8",       // IPv4-mapped public
	}

	for _, ipStr := range publicIPs {
		t.Run(ipStr, func(t *testing.T) {
			addr := ParseIP(ipStr)
			simdResult := addr.IsPublic()
			scalarResult := isPublicScalar(net.ParseIP(ipStr))

			assert.Equal(t, scalarResult, simdResult,
				"SIMD result mismatch for public IP %s (SIMD=%v, Scalar=%v)",
				ipStr, simdResult, scalarResult)
			assert.True(t, simdResult, "Expected %s to be public", ipStr)
		})
	}
}

// TestSIMDCorrectness_PrivateIPs verifies that SIMD implementation matches
// scalar implementation for known private IP addresses.
func TestSIMDCorrectness_PrivateIPs(t *testing.T) {
	privateIPs := []string{
		// RFC1918 addresses
		"10.0.0.1",
		"10.127.127.1",
		"10.255.255.254",
		"172.16.0.1",
		"172.31.254.254",
		"192.168.0.1",
		"192.168.255.255",
		// Other reserved ranges
		"100.64.0.1",    // Shared address space
		"100.127.255.1", // Shared address space
		"169.254.0.1",   // Link-local
		"169.254.255.1", // Link-local
		// IPv6 private
		"fd00::1",
		"fd12:3456:789a:1::1",
		"fe80::1",
		// IPv4-mapped private
		"::ffff:10.1.1.1",
		"::ffff:192.168.1.1",
	}

	for _, ipStr := range privateIPs {
		t.Run(ipStr, func(t *testing.T) {
			addr := ParseIP(ipStr)
			simdResult := addr.IsPublic()
			scalarResult := isPublicScalar(net.ParseIP(ipStr))

			assert.Equal(t, scalarResult, simdResult,
				"SIMD result mismatch for private IP %s (SIMD=%v, Scalar=%v)",
				ipStr, simdResult, scalarResult)
			assert.False(t, simdResult, "Expected %s to be private", ipStr)
		})
	}
}

// TestSIMDCorrectness_EdgeCases tests boundary conditions and edge cases.
func TestSIMDCorrectness_EdgeCases(t *testing.T) {
	edgeCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Boundary of 10.0.0.0/8
		{"10_lower_bound", "10.0.0.0", false},
		{"10_upper_bound", "10.255.255.255", false},
		{"before_10", "9.255.255.255", true},
		{"after_10", "11.0.0.0", true},

		// Boundary of 172.16.0.0/12
		{"172_16_lower", "172.16.0.0", false},
		{"172_31_upper", "172.31.255.255", false},
		{"before_172_16", "172.15.255.255", true},
		{"after_172_31", "172.32.0.0", true},

		// Boundary of 192.168.0.0/16
		{"192_168_lower", "192.168.0.0", false},
		{"192_168_upper", "192.168.255.255", false},
		{"192_167", "192.167.255.255", true},
		{"192_169", "192.169.0.0", true},

		// Boundary of 100.64.0.0/10
		{"100_64_lower", "100.64.0.0", false},
		{"100_127_upper", "100.127.255.255", false},
		{"100_63", "100.63.255.255", true},
		{"100_128", "100.128.0.0", true},

		// Boundary of 169.254.0.0/16
		{"169_254_lower", "169.254.0.0", false},
		{"169_254_upper", "169.254.255.255", false},
		{"169_253", "169.253.255.255", true},
		{"169_255", "169.255.0.0", true},

		// Special addresses
		{"all_zeros", "0.0.0.0", true},
		{"all_ones", "255.255.255.255", true},
		{"localhost", "127.0.0.1", true}, // Loopback is considered public by IsPublic()
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			addr := ParseIP(tc.ip)
			simdResult := addr.IsPublic()
			scalarResult := isPublicScalar(net.ParseIP(tc.ip))

			assert.Equal(t, scalarResult, simdResult,
				"SIMD result mismatch for %s (%s): SIMD=%v, Scalar=%v",
				tc.name, tc.ip, simdResult, scalarResult)
			assert.Equal(t, tc.expected, simdResult,
				"Expected %s (%s) to be public=%v, got %v",
				tc.name, tc.ip, tc.expected, simdResult)
		})
	}
}

// TestSIMDDirectComparison directly tests the SIMD utility function against scalar.
func TestSIMDDirectComparison(t *testing.T) {
	testCases := []struct {
		ip       [4]byte
		expected bool
	}{
		{[4]byte{8, 8, 8, 8}, true},         // Public
		{[4]byte{10, 1, 2, 3}, false},       // 10.0.0.0/8
		{[4]byte{172, 16, 0, 1}, false},     // 172.16.0.0/12
		{[4]byte{192, 168, 1, 1}, false},    // 192.168.0.0/16
		{[4]byte{100, 64, 0, 1}, false},     // 100.64.0.0/10
		{[4]byte{169, 254, 1, 1}, false},    // 169.254.0.0/16
		{[4]byte{127, 0, 0, 1}, true},       // Localhost (public per IsPublic semantics)
		{[4]byte{0, 0, 0, 0}, true},         // All zeros
		{[4]byte{255, 255, 255, 255}, true}, // All ones
	}

	for _, tc := range testCases {
		t.Run(net.IP(tc.ip[:]).String(), func(t *testing.T) {
			simdResult := simdutil.CheckIPv4Public(tc.ip)
			scalarResult := isPublicIPv4Scalar(net.IP(tc.ip[:]))

			assert.Equal(t, scalarResult, simdResult,
				"SIMD mismatch for %v: SIMD=%v, Scalar=%v",
				tc.ip, simdResult, scalarResult)
			assert.Equal(t, tc.expected, simdResult,
				"Expected %v to be public=%v, got %v",
				tc.ip, tc.expected, simdResult)
		})
	}
}

// FuzzIsPublicSIMD performs fuzz testing to compare SIMD vs scalar implementation.
// This test generates random IPv4 addresses and verifies that SIMD and scalar
// implementations always produce identical results.
func FuzzIsPublicSIMD(f *testing.F) {
	// Seed corpus with interesting values
	f.Add(byte(8), byte(8), byte(8), byte(8))         // Public
	f.Add(byte(10), byte(0), byte(0), byte(1))        // 10.x
	f.Add(byte(172), byte(16), byte(0), byte(1))      // 172.16.x
	f.Add(byte(192), byte(168), byte(0), byte(1))     // 192.168.x
	f.Add(byte(100), byte(64), byte(0), byte(1))      // 100.64.x
	f.Add(byte(169), byte(254), byte(0), byte(1))     // 169.254.x
	f.Add(byte(0), byte(0), byte(0), byte(0))         // Edge case
	f.Add(byte(255), byte(255), byte(255), byte(255)) // Edge case

	f.Fuzz(func(t *testing.T, a, b, c, d byte) {
		ip := net.IPv4(a, b, c, d)
		addr := FromNetIP(ip)

		simdResult := addr.IsPublic()
		scalarResult := isPublicScalar(ip)

		if simdResult != scalarResult {
			t.Errorf("Mismatch for %v: SIMD=%v, Scalar=%v",
				ip, simdResult, scalarResult)
		}
	})
}

// isPublicScalar is a helper that calls the appropriate scalar function based on IP type.
func isPublicScalar(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.To4() != nil {
		return isPublicIPv4Scalar(ip)
	}
	return isPublicIPv6Scalar(ip)
}
