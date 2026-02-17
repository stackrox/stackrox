package net

import (
	"math/rand"
	"strconv"
	"testing"
)

// BenchmarkIsPublic benchmarks individual IP classification
func BenchmarkIsPublic(b *testing.B) {
	testCases := []struct {
		name string
		ip   string
	}{
		// Public IPs (worst case - checks all 5 networks)
		{"PublicIPv4", "8.8.8.8"},
		{"PublicIPv4_AWS", "54.239.28.85"},

		// Private IPs (varying early-exit positions)
		{"PrivateIPv4_10", "10.1.2.3"},
		{"PrivateIPv4_192", "192.168.1.1"},
		{"PrivateIPv4_172", "172.16.0.1"},
		{"PrivateIPv4_100", "100.64.0.1"},
		{"PrivateIPv4_169", "169.254.1.1"},

		// IPv6
		{"PublicIPv6", "2001:4860:4860::8888"},
		{"PrivateIPv6_ULA", "fd00::1"},
		{"PrivateIPv6_LinkLocal", "fe80::1"},

		// IPv4-mapped IPv6
		{"IPv4Mapped_Public", "::ffff:8.8.8.8"},
		{"IPv4Mapped_Private", "::ffff:10.1.1.1"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			addr := ParseIP(tc.ip)
			for b.Loop() {
				_ = addr.IsPublic()
			}
		})
	}
}

// BenchmarkIsPublicBatch simulates network flow manager workload
func BenchmarkIsPublicBatch(b *testing.B) {
	ips := generateMixedIPs(1000)
	for b.Loop() {
		for _, ip := range ips {
			_ = ip.IsPublic()
		}
	}
}

// BenchmarkIsPublicWorstCase: all public IPs (no early exit)
func BenchmarkIsPublicWorstCase(b *testing.B) {
	ips := generatePublicIPs(1000)
	for b.Loop() {
		for _, ip := range ips {
			_ = ip.IsPublic()
		}
	}
}

// Helper functions
func generateMixedIPs(count int) []IPAddress {
	ips := make([]IPAddress, count)
	for i := range count {
		if i%2 == 0 {
			ips[i] = ParseIP(randomPublicIPv4())
		} else {
			ips[i] = ParseIP(randomPrivateIPv4())
		}
	}
	return ips
}

func generatePublicIPs(count int) []IPAddress {
	ips := make([]IPAddress, count)
	for i := range count {
		ips[i] = ParseIP(randomPublicIPv4())
	}
	return ips
}

func randomPublicIPv4() string {
	return "8." + randOctet() + "." + randOctet() + "." + randOctet()
}

func randomPrivateIPv4() string {
	switch rand.Intn(3) {
	case 0:
		return "10." + randOctet() + "." + randOctet() + "." + randOctet()
	case 1:
		return "192.168." + randOctet() + "." + randOctet()
	default:
		return "172.16." + randOctet() + "." + randOctet()
	}
}

func randOctet() string {
	return strconv.Itoa(rand.Intn(256))
}
