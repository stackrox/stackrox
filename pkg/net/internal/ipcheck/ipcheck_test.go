package ipcheck

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIPv4Public(t *testing.T) {
	tests := []struct {
		name     string
		ip       [4]byte
		expected bool
	}{
		// Public IPs
		{"Google DNS", [4]byte{8, 8, 8, 8}, true},
		{"Cloudflare", [4]byte{1, 1, 1, 1}, true},
		{"AWS", [4]byte{54, 239, 28, 85}, true},

		// Private IPs - 10.0.0.0/8
		{"10.0.0.1", [4]byte{10, 0, 0, 1}, false},
		{"10.255.255.254", [4]byte{10, 255, 255, 254}, false},

		// Private IPs - 192.168.0.0/16
		{"192.168.0.1", [4]byte{192, 168, 0, 1}, false},
		{"192.168.255.255", [4]byte{192, 168, 255, 255}, false},

		// Private IPs - 172.16.0.0/12
		{"172.16.0.1", [4]byte{172, 16, 0, 1}, false},
		{"172.31.255.254", [4]byte{172, 31, 255, 254}, false},

		// Private IPs - 100.64.0.0/10
		{"100.64.0.1", [4]byte{100, 64, 0, 1}, false},
		{"100.127.255.254", [4]byte{100, 127, 255, 254}, false},

		// Private IPs - 169.254.0.0/16
		{"169.254.0.1", [4]byte{169, 254, 0, 1}, false},
		{"169.254.255.254", [4]byte{169, 254, 255, 254}, false},

		// Boundary testing
		{"9.255.255.255 (before 10.x)", [4]byte{9, 255, 255, 255}, true},
		{"11.0.0.0 (after 10.x)", [4]byte{11, 0, 0, 0}, true},
		{"172.15.255.255 (before 172.16.x)", [4]byte{172, 15, 255, 255}, true},
		{"172.32.0.0 (after 172.31.x)", [4]byte{172, 32, 0, 0}, true},

		// Special addresses
		{"127.0.0.1 (localhost)", [4]byte{127, 0, 0, 1}, true}, // Loopback is public per isPublic semantics
		{"0.0.0.0", [4]byte{0, 0, 0, 0}, true},
		{"255.255.255.255", [4]byte{255, 255, 255, 255}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIPv4Public(tt.ip)
			assert.Equal(t, tt.expected, result,
				"IsIPv4Public(%v) = %v, expected %v",
				net.IP(tt.ip[:]), result, tt.expected)
		})
	}
}

func TestIsIPv6Public(t *testing.T) {
	tests := []struct {
		name     string
		ipStr    string
		expected bool
	}{
		// Public IPv6
		{"Google DNS", "2001:4860:4860::8888", true},
		{"Cloudflare", "2606:4700:4700::1111", true},

		// Private IPv6 - ULA (fd00::/8)
		{"ULA", "fd00::1", false},
		{"ULA with data", "fd12:3456:789a:1::1", false},

		// Private IPv6 - Link-Local (fe80::/10)
		{"Link-Local", "fe80::1", false},
		{"Link-Local with data", "fe80::250:56ff:fe9a:8f73", false},

		// IPv4-mapped IPv6
		{"IPv4-mapped public", "::ffff:8.8.8.8", true},
		{"IPv4-mapped private 10.x", "::ffff:10.1.1.1", false},
		{"IPv4-mapped private 192.168.x", "::ffff:192.168.1.1", false},

		// IPv6 loopback (::1)
		{"Loopback", "::1", true}, // Loopback is public per isPublic semantics
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ipStr)
			var ipv6 [16]byte
			copy(ipv6[:], ip.To16())

			result := IsIPv6Public(ipv6)
			assert.Equal(t, tt.expected, result,
				"IsIPv6Public(%s) = %v, expected %v",
				tt.ipStr, result, tt.expected)
		})
	}
}
