package net

import (
	"testing"

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
