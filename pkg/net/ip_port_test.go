package net

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumericEndpointV4(t *testing.T) {
	ep := NetworkPeerID{
		Address: IPAddress{data: ipv4data{192, 168, 0, 1}},
		Port:    1234,
	}
	assert.True(t, ep.IsAddressValid())
	assert.Equal(t, "192.168.0.1:1234", ep.String())
}

func TestNumericEndpointV4NoPort(t *testing.T) {
	ep := NetworkPeerID{
		Address: IPAddress{data: ipv4data{192, 168, 0, 1}},
	}
	assert.True(t, ep.IsAddressValid())
	assert.Equal(t, "192.168.0.1", ep.String())
}

func TestNumericEndpointV6(t *testing.T) {
	ep := NetworkPeerID{
		Address: IPAddress{data: ipv6data{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		Port:    1234,
	}
	assert.True(t, ep.IsAddressValid())
	assert.Equal(t, "[::1]:1234", ep.String())
}

func TestNumericEndpointV6NoPort(t *testing.T) {
	ep := NetworkPeerID{
		Address: IPAddress{data: ipv6data{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
	}
	assert.True(t, ep.IsAddressValid())
	assert.Equal(t, "::1", ep.String())
}

func TestParseNumericEndpointV4(t *testing.T) {
	ep := ParseIPPortPair("192.168.0.1:1234")
	expected := NetworkPeerID{
		Address: IPAddress{data: ipv4data{192, 168, 0, 1}},
		Port:    1234,
	}
	assert.Equal(t, expected, ep)
}

func TestParseNumericEndpointV4NoPort(t *testing.T) {
	ep := ParseIPPortPair("192.168.0.1")
	expected := NetworkPeerID{
		Address: IPAddress{data: ipv4data{192, 168, 0, 1}},
	}
	assert.Equal(t, expected, ep)
}

func TestParseNumericEndpointV6(t *testing.T) {
	ep := ParseIPPortPair("[::1]:1234")
	expected := NetworkPeerID{
		Address: IPAddress{data: ipv6data{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		Port:    1234,
	}
	assert.Equal(t, expected, ep)
}

func TestParseNumericEndpointV6NoPort(t *testing.T) {
	ep := ParseIPPortPair("::1")
	expected := NetworkPeerID{
		Address: IPAddress{data: ipv6data{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
	}
	assert.Equal(t, expected, ep)
}

func TestParseNumericEndpointInvalid(t *testing.T) {
	ep := ParseIPPortPair("hostname:1234")
	assert.False(t, ep.IsAddressValid())

	ep = ParseIPPortPair("192.168.0.1:port")
	assert.False(t, ep.IsAddressValid())
}

func TestNetworkPeerID_String(t *testing.T) {
	tests := map[string]struct {
		Address   IPAddress
		Port      uint16
		IPNetwork IPNetwork
		want      string
	}{
		"IPv4 address": {
			Address:   IPAddress{data: ipv4data{192, 168, 0, 1}},
			Port:      80,
			IPNetwork: IPNetwork{},
			want:      "192.168.0.1:80",
		},
		"IPv4 address without port": {
			Address:   IPAddress{data: ipv4data{192, 168, 0, 1}},
			Port:      0,
			IPNetwork: IPNetwork{},
			want:      "192.168.0.1",
		},
		"IPv4 network address": {
			Address: IPAddress{},
			Port:    80,
			IPNetwork: IPNetwork{
				ip:        IPAddress{data: ipv4data{192, 168, 0, 0}},
				prefixLen: 24,
			},
			want: "192.168.0.0/24:80",
		},
		"IPv6 address": {
			Address:   IPAddress{data: ipv6data{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
			Port:      80,
			IPNetwork: IPNetwork{},
			want:      "[::1]:80",
		},
		"IPv6 network address with prefix": {
			Address: IPAddress{},
			Port:    80,
			IPNetwork: IPNetwork{
				ip:        IPAddress{data: ipv6data{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
				prefixLen: 128,
			},
			want: "[::1/128]:80",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			e := NetworkPeerID{
				Address:   tt.Address,
				Port:      tt.Port,
				IPNetwork: tt.IPNetwork,
			}
			assert.Equalf(t, tt.want, e.String(), "String()")
		})
	}
}

func FuzzParseIPPortPair(f *testing.F) {
	// Seed corpus with valid and invalid IP:port pairs
	seeds := []string{
		"192.168.0.1:1234",
		"10.0.0.1:80",
		"127.0.0.1:8080",
		"0.0.0.0:0",
		"255.255.255.255:65535",
		"[::1]:1234",
		"[2001:db8::1]:80",
		"[fe80::1]:8080",
		"[::ffff:192.168.0.1]:443",
		"192.168.0.1",        // No port
		"::1",                // IPv6 no port
		"",                   // Empty
		"hostname:1234",      // Invalid hostname
		"192.168.0.1:port",   // Invalid port
		"192.168.0.1:-1",     // Negative port
		"192.168.0.1:65536",  // Port too large
		"192.168.0.1:0",      // Port zero (invalid)
		"256.0.0.1:80",       // Invalid IP
		"1.2.3.4.5:80",       // Invalid IP
		"[::1:80",            // Missing closing bracket
		"::1]:80",            // Missing opening bracket
		"[::1:80]",           // Malformed IPv6 with port
		":1234",              // Missing IP
		"192.168.0.1:",       // Missing port number
		"[]:1234",            // Empty IPv6
		"192.168.0.1:abc",    // Non-numeric port
		"192.168.0.1:99999",  // Port way too large
		"[::gggg]:80",        // Invalid IPv6
		"localhost:80",       // Hostname not allowed
		"example.com:443",    // Domain name not allowed
		"192.168.1.1:1",      // Valid minimum port
		"::ffff:10.0.0.1:22", // IPv4-mapped without brackets
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, input string) {
		peer := ParseIPPortPair(input)
		if peer.IsAddressValid() {
			_ = peer.Address.String()
			_ = peer.Port
			_ = peer.String()
		}
	})
}
