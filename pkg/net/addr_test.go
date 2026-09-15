package net

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPv4(t *testing.T) {
	addr := IPAddress{
		data: ipv4data{192, 168, 0, 1},
	}
	assert.True(t, addr.IsValid())
	assert.Equal(t, "192.168.0.1", addr.String())
	assert.Equal(t, IPv4, addr.Family())
	assert.True(t, addr.AsNetIP().Equal(net.ParseIP("192.168.0.1")))
}

func TestIPv6(t *testing.T) {
	addr := IPAddress{
		data: ipv6data{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
	}
	assert.True(t, addr.IsValid())
	assert.Equal(t, "::1", addr.String())
	assert.Equal(t, IPv6, addr.Family())
	assert.True(t, addr.AsNetIP().Equal(net.ParseIP("::1")))
}

func TestInvalid(t *testing.T) {
	addr := IPAddress{}
	assert.False(t, addr.IsValid())
	assert.Empty(t, addr.String())
	assert.Equal(t, InvalidFamily, addr.Family())
	assert.Nil(t, addr.AsNetIP())
}

func TestParseIPv4(t *testing.T) {
	addr := ParseIP("192.168.0.1")
	assert.Equal(t, IPAddress{data: ipv4data{192, 168, 0, 1}}, addr)
}

func TestParseIPv6(t *testing.T) {
	addr := ParseIP("::1")
	assert.Equal(t, IPAddress{data: ipv6data{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}, addr)
}

func TestParseInvalid(t *testing.T) {
	addr := ParseIP("1.2.3.4.5")
	assert.False(t, addr.IsValid())
}

func TestIPv4FromBytes(t *testing.T) {
	addr := IPFromBytes([]byte{192, 168, 0, 1})
	assert.Equal(t, IPAddress{data: ipv4data{192, 168, 0, 1}}, addr)
}

func TestIPv6FromBytes(t *testing.T) {
	addr := IPFromBytes([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	assert.Equal(t, IPAddress{data: ipv6data{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}, addr)
}

func TestIPv4MappedIPv6FromBytes(t *testing.T) {
	addr := IPFromBytes([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x0A, 0x14, 0x25, 0xB2})
	assert.Equal(t, IPAddress{data: ipv4data{10, 20, 37, 178}}, addr)
}

func TestIsPublic_True(t *testing.T) {

	publicIPs := []string{
		"4.4.4.4",
		"8.8.8.8",
		"131.217.0.129",
		"127.0.0.1",
		"::1",
		"2a02:908:e850:cf20:9919:44af:a46e:1669",
		"::ffff:4.4.4.4",
	}

	for _, publicIP := range publicIPs {
		ip := ParseIP(publicIP)
		assert.True(t, ip.IsPublic(), "expected IP %s to be public", publicIP)
	}
}

func TestIsPublic_False(t *testing.T) {

	privateIPs := []string{
		"10.127.127.1",
		"172.31.254.254",
		"192.168.0.1",
		"fd12:3456:789a:1::1",
		"::ffff:10.1.1.1",
	}

	for _, privateIP := range privateIPs {
		ip := ParseIP(privateIP)
		assert.False(t, ip.IsPublic(), "expected IP %s to be private", privateIP)
	}
}

func TestFromCIDRString_Valid(t *testing.T) {

	cidrs := []string{
		"192.168.0.1/8",
		"0.0.0.0/0",
		"::ffff:4.4.4.4/32",
		"::ffff:4.4.4.4/0",
		"::1/52",
	}

	for _, cidr := range cidrs {
		actual := IPNetworkFromCIDR(cidr)
		assert.NotEqual(t, IPNetwork{}, actual)
	}
}

func TestFromCIDRString_InValid(t *testing.T) {

	cidrs := []string{
		"192.168.0.1/64",
		"0.0.0.0",
		"::ffff:4.4.4.4/200",
		"::ffff:.4/0",
		"::1",
	}

	for _, cidr := range cidrs {
		actual := IPNetworkFromCIDR(cidr)
		assert.Equal(t, IPNetwork{}, actual)
	}
}

func TestFromCIDRBytes_Valid(t *testing.T) {

	cidrs := [][]byte{
		{192, 168, 0, 1, 8},
		{0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 4, 4, 4, 4, 0},
	}

	for _, cidr := range cidrs {
		actual := IPNetworkFromCIDRBytes(cidr)
		assert.NotEqual(t, IPNetwork{}, actual)
	}
}

func TestFromCIDRBytesInvalid(t *testing.T) {

	cidrs := [][]byte{
		{192, 168, 0, 1, 64},
		{0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 200},
		{0, 0, 255, 0, 4, 0},
		{0, 0, 1},
	}

	for _, cidr := range cidrs {
		actual := IPNetworkFromCIDRBytes(cidr)
		assert.Equal(t, IPNetwork{}, actual)
	}
}

func TestIPAddressCompare(t *testing.T) {
	tests := map[string]struct {
		a        IPAddress
		b        IPAddress
		expected int
	}{
		"should return zero when addresses are identical": {
			a:        ParseIP("192.168.1.1"),
			b:        ParseIP("192.168.1.1"),
			expected: 0,
		},
		"should return negative when first address is lexicographically smaller": {
			a:        ParseIP("192.168.1.1"),
			b:        ParseIP("192.168.1.2"),
			expected: -1,
		},
		"should return positive when first address is lexicographically larger": {
			a:        ParseIP("192.168.1.2"),
			b:        ParseIP("192.168.1.1"),
			expected: 1,
		},
		"should return negative when IPv4 compared to IPv6 due to byte length": {
			a:        ParseIP("192.168.1.1"),
			b:        ParseIP("::1"),
			expected: -12,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := IPAddressCompare(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func FuzzParseIP(f *testing.F) {
	// Seed corpus with valid and invalid IP addresses
	seeds := []string{
		"192.168.0.1",
		"10.0.0.1",
		"172.16.0.1",
		"127.0.0.1",
		"0.0.0.0",
		"255.255.255.255",
		"::1",
		"::ffff:192.168.0.1",
		"2001:db8::1",
		"fe80::1",
		"2a02:908:e850:cf20:9919:44af:a46e:1669",
		"",
		"invalid",
		"1.2.3.4.5",
		"256.1.1.1",
		"a.b.c.d",
		"::::",
		"gggg::1",
		"192.168.1",
		"192.168.1.1.1",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, input string) {
		addr := ParseIP(input)
		if addr.IsValid() {
			_ = addr.String()
			_ = addr.Family()
			_ = addr.AsNetIP()
		}
	})
}

func FuzzIPFromBytes(f *testing.F) {
	// Seed corpus with various byte slices
	seeds := [][]byte{
		{192, 168, 0, 1},     // IPv4
		{10, 0, 0, 1},        // IPv4
		{127, 0, 0, 1},       // IPv4 loopback
		{0, 0, 0, 0},         // IPv4 unspecified
		{255, 255, 255, 255}, // IPv4 broadcast
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},                                         // IPv6 loopback
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                                         // IPv6 unspecified
		{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},                             // IPv6
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 192, 168, 0, 1}, // IPv4-mapped IPv6
		{},                          // Empty
		{1},                         // Too short
		{1, 2},                      // Too short
		{1, 2, 3},                   // Too short
		{1, 2, 3, 4, 5},             // Between IPv4 and IPv6
		{1, 2, 3, 4, 5, 6, 7, 8, 9}, // Too short for IPv6
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, input []byte) {
		addr := IPFromBytes(input)
		if addr.IsValid() {
			_ = addr.String()
			_ = addr.Family()
			_ = addr.AsNetIP()
		}
	})
}

func FuzzIPNetworkFromCIDR(f *testing.F) {
	// Seed corpus with valid and invalid CIDR strings
	seeds := []string{
		"192.168.0.0/24",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"0.0.0.0/0",
		"192.168.1.1/32",
		"::1/128",
		"2001:db8::/32",
		"fe80::/10",
		"::ffff:192.168.0.0/96",
		"::/0",
		"",
		"192.168.0.1",        // Missing prefix
		"192.168.0.0/",       // Missing prefix length
		"192.168.0.0/33",     // Invalid prefix for IPv4
		"192.168.0.0/-1",     // Negative prefix
		"192.168.0.0/abc",    // Non-numeric prefix
		"invalid/24",         // Invalid IP
		"::1/129",            // Invalid prefix for IPv6
		"256.0.0.0/24",       // Invalid IP octet
		"/24",                // Missing IP
		"192.168.0.0/24/24",  // Multiple slashes
		"::1/52",             // Valid IPv6 with prefix
		"::ffff:4.4.4.4/32",  // Valid IPv4-mapped
		"::ffff:4.4.4.4/200", // Invalid prefix
		"::ffff:.4/0",        // Invalid IP part
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, input string) {
		network := IPNetworkFromCIDR(input)
		if network.IsValid() {
			_ = network.IP()
			_ = network.Family()
			_ = network.PrefixLen()
			_ = network.AsIPNet()
			_ = network.String()
			_ = network.Contains(network.IP())
		} else {
			_ = network.IP()
			_ = network.Family()
			_ = network.String()
			_ = network.Contains(ParseIP("192.168.0.1"))
		}
	})
}
