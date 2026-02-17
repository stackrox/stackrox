package simdutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCheckIPv4Public verifies that the IP checking function works correctly.
func TestCheckIPv4Public(t *testing.T) {
	testCases := []struct {
		name     string
		ip       [4]byte
		expected bool
	}{
		// Public IPs
		{"Google DNS", [4]byte{8, 8, 8, 8}, true},
		{"Cloudflare DNS", [4]byte{1, 1, 1, 1}, true},
		{"Random public", [4]byte{54, 239, 28, 85}, true},

		// Private IPs - 10.0.0.0/8
		{"10.0.0.1", [4]byte{10, 0, 0, 1}, false},
		{"10.127.127.1", [4]byte{10, 127, 127, 1}, false},
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

		// Edge cases
		{"0.0.0.0", [4]byte{0, 0, 0, 0}, true},
		{"255.255.255.255", [4]byte{255, 255, 255, 255}, true},
		{"127.0.0.1 (localhost)", [4]byte{127, 0, 0, 1}, true},

		// Boundary testing
		{"9.255.255.255 (before 10.x)", [4]byte{9, 255, 255, 255}, true},
		{"11.0.0.0 (after 10.x)", [4]byte{11, 0, 0, 0}, true},
		{"172.15.255.255 (before 172.16.x)", [4]byte{172, 15, 255, 255}, true},
		{"172.32.0.0 (after 172.31.x)", [4]byte{172, 32, 0, 0}, true},
		{"192.167.255.255 (before 192.168.x)", [4]byte{192, 167, 255, 255}, true},
		{"192.169.0.0 (after 192.168.x)", [4]byte{192, 169, 0, 0}, true},
		{"100.63.255.255 (before 100.64.x)", [4]byte{100, 63, 255, 255}, true},
		{"100.128.0.0 (after 100.127.x)", [4]byte{100, 128, 0, 0}, true},
		{"169.253.255.255 (before 169.254.x)", [4]byte{169, 253, 255, 255}, true},
		{"169.255.0.0 (after 169.254.x)", [4]byte{169, 255, 0, 0}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CheckIPv4Public(tc.ip)
			assert.Equal(t, tc.expected, result,
				"CheckIPv4Public(%v) = %v, expected %v",
				tc.ip, result, tc.expected)
		})
	}
}
