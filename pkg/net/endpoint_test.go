package net

import (
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/assert"
)

func TestNumericEndpointCompare(t *testing.T) {
	tests := map[string]struct {
		a        NumericEndpoint
		b        NumericEndpoint
		expected int
	}{
		"should return zero when endpoints are identical": {
			a:        MakeNumericEndpoint(ParseIP("192.168.1.1"), 80, TCP),
			b:        MakeNumericEndpoint(ParseIP("192.168.1.1"), 80, TCP),
			expected: 0,
		},
		"should return negative when first IP address is smaller": {
			a:        MakeNumericEndpoint(ParseIP("192.168.1.1"), 80, TCP),
			b:        MakeNumericEndpoint(ParseIP("192.168.1.2"), 80, TCP),
			expected: -1,
		},
		"should return negative when IPs are equal but first port is smaller": {
			a:        MakeNumericEndpoint(ParseIP("192.168.1.1"), 80, TCP),
			b:        MakeNumericEndpoint(ParseIP("192.168.1.1"), 443, TCP),
			expected: -363,
		},
		"should return negative when IPs and ports are equal but first protocol is smaller": {
			a:        MakeNumericEndpoint(ParseIP("192.168.1.1"), 80, TCP),
			b:        MakeNumericEndpoint(ParseIP("192.168.1.1"), 80, UDP),
			expected: -1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := NumericEndpointCompare(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNumericEndpointBinaryKey(t *testing.T) {
	h := xxhash.New()
	var buf [16]byte

	tests := map[string]struct {
		ep1       NumericEndpoint
		ep2       NumericEndpoint
		shouldEq  bool
		assertion string
	}{
		"identical endpoints produce same hash": {
			ep1:       MakeNumericEndpoint(ParseIP("10.0.0.1"), 8080, TCP),
			ep2:       MakeNumericEndpoint(ParseIP("10.0.0.1"), 8080, TCP),
			shouldEq:  true,
			assertion: "same endpoints should produce same hash",
		},
		"different IP produces different hash": {
			ep1:       MakeNumericEndpoint(ParseIP("10.0.0.1"), 8080, TCP),
			ep2:       MakeNumericEndpoint(ParseIP("10.0.0.2"), 8080, TCP),
			shouldEq:  false,
			assertion: "different IPs should produce different hashes",
		},
		"different port produces different hash": {
			ep1:       MakeNumericEndpoint(ParseIP("10.0.0.1"), 8080, TCP),
			ep2:       MakeNumericEndpoint(ParseIP("10.0.0.1"), 8081, TCP),
			shouldEq:  false,
			assertion: "different ports should produce different hashes",
		},
		"different protocol produces different hash": {
			ep1:       MakeNumericEndpoint(ParseIP("10.0.0.1"), 8080, TCP),
			ep2:       MakeNumericEndpoint(ParseIP("10.0.0.1"), 8080, UDP),
			shouldEq:  false,
			assertion: "different protocols should produce different hashes",
		},
		"IPv6 endpoints hash consistently": {
			ep1:       MakeNumericEndpoint(ParseIP("2001:db8::1"), 443, TCP),
			ep2:       MakeNumericEndpoint(ParseIP("2001:db8::1"), 443, TCP),
			shouldEq:  true,
			assertion: "IPv6 endpoints should hash consistently",
		},
		"IPv4-mapped IPv6 normalizes to IPv4 hash": {
			ep1:       MakeNumericEndpoint(ParseIP("192.168.1.1"), 80, TCP),
			ep2:       MakeNumericEndpoint(ParseIP("::ffff:192.168.1.1"), 80, TCP),
			shouldEq:  true,
			assertion: "IPv4-mapped IPv6 addresses are normalized to IPv4 and should have same hash",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hash1 := tt.ep1.BinaryKey(h, &buf)
			hash2 := tt.ep2.BinaryKey(h, &buf)

			if tt.shouldEq {
				assert.Equal(t, hash1, hash2, tt.assertion)
			} else {
				assert.NotEqual(t, hash1, hash2, tt.assertion)
			}

			// Verify hash is non-zero for valid endpoints
			if tt.ep1.IsValid() {
				assert.NotEqual(t, BinaryHash(0), hash1, "hash should be non-zero for valid endpoint")
			}
		})
	}
}

func TestBinaryKeyIsStable(t *testing.T) {
	// Verify that hashing the same endpoint multiple times produces the same hash
	h := xxhash.New()
	var buf [16]byte
	ep := MakeNumericEndpoint(ParseIP("10.0.0.1"), 8080, TCP)

	hash1 := ep.BinaryKey(h, &buf)
	hash2 := ep.BinaryKey(h, &buf)
	hash3 := ep.BinaryKey(h, &buf)

	assert.Equal(t, hash1, hash2, "hash should be stable across calls")
	assert.Equal(t, hash2, hash3, "hash should be stable across calls")
}
