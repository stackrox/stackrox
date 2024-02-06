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
