package netutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIPNetSubnet_Equal(t *testing.T) {
	t.Parallel()

	net1 := MustParseCIDR("127.0.0.0/8")
	net2 := MustParseCIDR("127.0.0.0/8")
	assert.True(t, IsIPNetSubset(net1, net2))
}

func TestIsIPNetSubnet_Disjoint(t *testing.T) {
	t.Parallel()

	net1 := MustParseCIDR("127.0.0.0/8")
	net2 := MustParseCIDR("10.0.0.0/8")
	assert.False(t, IsIPNetSubset(net1, net2))
	assert.False(t, IsIPNetSubset(net2, net1))
}

func TestIsIPNetSubnet_Contained(t *testing.T) {
	t.Parallel()

	net1 := MustParseCIDR("127.0.0.0/8")
	net2 := MustParseCIDR("127.0.1.1/31")
	assert.True(t, IsIPNetSubset(net1, net2))
	assert.False(t, IsIPNetSubset(net2, net1))
}
