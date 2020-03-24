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
	t.Parallel()

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
	t.Parallel()

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
