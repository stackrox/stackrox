package net

import (
	"math/rand"
	"strconv"
	"testing"
)

// BenchmarkIsPublic benchmarks the IsPublic() method for various IP addresses.
// This measures the performance of individual IP classification calls.
func BenchmarkIsPublic(b *testing.B) {
	testCases := []struct {
		name string
		ip   string
	}{
		// Public IPs (worst case - must check all 5 private networks)
		{"PublicIPv4", "8.8.8.8"},
		{"PublicIPv4_AWS", "54.239.28.85"},

		// Private IPs (varying early-exit positions)
		{"PrivateIPv4_10", "10.1.2.3"},     // Matches 1st network (10.0.0.0/8)
		{"PrivateIPv4_192", "192.168.1.1"}, // Matches 2nd network (192.168.0.0/16)
		{"PrivateIPv4_172", "172.16.0.1"},  // Matches 3rd network (172.16.0.0/12)
		{"PrivateIPv4_100", "100.64.0.1"},  // Matches 4th network (100.64.0.0/10)
		{"PrivateIPv4_169", "169.254.1.1"}, // Matches 5th network (169.254.0.0/16)

		// IPv6 addresses
		{"PublicIPv6", "2001:4860:4860::8888"},
		{"PrivateIPv6_ULA", "fd00::1"},
		{"PrivateIPv6_LinkLocal", "fe80::1"},

		// IPv4-mapped IPv6 addresses
		{"IPv4Mapped_Public", "::ffff:8.8.8.8"},
		{"IPv4Mapped_Private", "::ffff:10.1.1.1"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			addr := ParseIP(tc.ip)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = addr.IsPublic()
			}
		})
	}
}

// BenchmarkIsPublicBatch simulates the network flow manager workload.
// It processes a batch of 1000 IPs (50% public, 50% private) which is
// representative of real-world sensor traffic during enrichment cycles.
func BenchmarkIsPublicBatch(b *testing.B) {
	ips := generateMixedIPs(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ip := range ips {
			_ = ip.IsPublic()
		}
	}
}

// BenchmarkIsPublicWorstCase measures performance when all IPs are public.
// This represents the worst case where we must check all 5 private networks
// for each IP (no early exit). SIMD optimization should show maximum benefit here.
func BenchmarkIsPublicWorstCase(b *testing.B) {
	ips := generatePublicIPs(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ip := range ips {
			_ = ip.IsPublic()
		}
	}
}

// BenchmarkIsPublicBestCase measures performance when all IPs match the first
// private network (10.0.0.0/8). This represents the best case with immediate
// early exit. SIMD may have less benefit here due to early termination.
func BenchmarkIsPublicBestCase(b *testing.B) {
	ips := generatePrivateIPs(1000, "10.")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ip := range ips {
			_ = ip.IsPublic()
		}
	}
}

// BenchmarkIsPublicMixedPrivate measures performance with a mix of different
// private network ranges (10.x, 172.16.x, 192.168.x) to test average-case behavior.
func BenchmarkIsPublicMixedPrivate(b *testing.B) {
	ips := generateMixedPrivateIPs(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ip := range ips {
			_ = ip.IsPublic()
		}
	}
}

// Helper functions to generate test data

// generateMixedIPs creates a slice of IP addresses with 50% public and 50% private.
func generateMixedIPs(count int) []IPAddress {
	ips := make([]IPAddress, count)
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			// Generate public IP (avoid private ranges)
			ips[i] = ParseIP(randomPublicIPv4())
		} else {
			// Generate private IP (mix of different ranges)
			ips[i] = ParseIP(randomPrivateIPv4())
		}
	}
	return ips
}

// generatePublicIPs creates a slice of only public IP addresses.
func generatePublicIPs(count int) []IPAddress {
	ips := make([]IPAddress, count)
	for i := 0; i < count; i++ {
		ips[i] = ParseIP(randomPublicIPv4())
	}
	return ips
}

// generatePrivateIPs creates a slice of private IPs from a specific range.
func generatePrivateIPs(count int, prefix string) []IPAddress {
	ips := make([]IPAddress, count)
	for i := 0; i < count; i++ {
		switch prefix {
		case "10.":
			ips[i] = ParseIP(randomIP("10."))
		case "192.168.":
			ips[i] = ParseIP(randomIP("192.168."))
		case "172.16.":
			ips[i] = ParseIP(randomIP("172.16."))
		default:
			ips[i] = ParseIP(randomIP("10."))
		}
	}
	return ips
}

// generateMixedPrivateIPs creates a mix of different private network ranges.
func generateMixedPrivateIPs(count int) []IPAddress {
	ips := make([]IPAddress, count)
	for i := 0; i < count; i++ {
		switch i % 3 {
		case 0:
			ips[i] = ParseIP(randomIP("10."))
		case 1:
			ips[i] = ParseIP(randomIP("192.168."))
		case 2:
			ips[i] = ParseIP(randomIP("172.16."))
		}
	}
	return ips
}

// randomPublicIPv4 generates a random public IPv4 address (not in private ranges).
func randomPublicIPv4() string {
	// Generate IPs in 8.0.0.0/8 range (Google Public DNS range, definitely public)
	return randomIP("8.")
}

// randomPrivateIPv4 generates a random private IPv4 address.
func randomPrivateIPv4() string {
	switch rand.Intn(3) {
	case 0:
		return randomIP("10.")
	case 1:
		return randomIP("192.168.")
	default:
		return randomIP("172.16.")
	}
}

// randomIP generates a random IP with the given prefix.
func randomIP(prefix string) string {
	switch prefix {
	case "10.":
		return "10." + randOctet() + "." + randOctet() + "." + randOctet()
	case "192.168.":
		return "192.168." + randOctet() + "." + randOctet()
	case "172.16.":
		return "172.16." + randOctet() + "." + randOctet()
	case "8.":
		return "8." + randOctet() + "." + randOctet() + "." + randOctet()
	default:
		return randOctet() + "." + randOctet() + "." + randOctet() + "." + randOctet()
	}
}

// randOctet generates a random IP octet (0-255) as a string.
func randOctet() string {
	return strconv.Itoa(rand.Intn(256))
}
