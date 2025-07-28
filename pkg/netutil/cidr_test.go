package netutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIPNetSubnet_Equal(t *testing.T) {

	net1 := MustParseCIDR("127.0.0.0/8")
	net2 := MustParseCIDR("127.0.0.0/8")
	assert.True(t, IsIPNetSubset(net1, net2))
}

func TestIsIPNetSubnet_Disjoint(t *testing.T) {

	net1 := MustParseCIDR("127.0.0.0/8")
	net2 := MustParseCIDR("10.0.0.0/8")
	assert.False(t, IsIPNetSubset(net1, net2))
	assert.False(t, IsIPNetSubset(net2, net1))
}

func TestIsIPNetSubnet_Contained(t *testing.T) {

	net1 := MustParseCIDR("127.0.0.0/8")
	net2 := MustParseCIDR("127.0.1.1/31")
	assert.True(t, IsIPNetSubset(net1, net2))
	assert.False(t, IsIPNetSubset(net2, net1))
}

func TestOverlap_Overlap(t *testing.T) {

	assert.True(t, Overlap(MustParseCIDR("172.16.0.0/16"), MustParseCIDR("172.16.0.0/24")))
}

func TestOverlap_NoOverlap(t *testing.T) {

	assert.False(t, Overlap(MustParseCIDR("127.16.0.0/16"), MustParseCIDR("172.16.0.0/24")))
}

func TestAnyOverlap_Overlap(t *testing.T) {

	nets := []*net.IPNet{MustParseCIDR("172.16.0.0/24"), MustParseCIDR("127.16.0.0/16")}
	assert.True(t, AnyOverlap(MustParseCIDR("172.16.0.0/16"), nets))
}

func TestAnyOverlap_NoOverlap(t *testing.T) {

	nets := []*net.IPNet{MustParseCIDR("172.16.0.0/24"), MustParseCIDR("127.16.0.0/16")}
	assert.False(t, AnyOverlap(MustParseCIDR("126.0.0.0/8"), nets))
}
