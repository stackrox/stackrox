package tree

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNRadixTreeIPv4(t *testing.T) {
	e1 := testutils.GetExtSrcNetworkEntityInfo("1", "1", "35.187.144.0/20", true)
	e2 := testutils.GetExtSrcNetworkEntityInfo("2", "2", "35.187.144.0/16", false)
	e3 := testutils.GetExtSrcNetworkEntityInfo("3", "3", "35.187.144.0/8", false)
	e4 := testutils.GetExtSrcNetworkEntityInfo("4", "4", "35.187.144.0/23", false)
	e5 := testutils.GetExtSrcNetworkEntityInfo("5", "5", "35.188.144.0/16", true)
	e6 := testutils.GetExtSrcNetworkEntityInfo("6", "6", "36.188.144.0/30", false)
	e7 := testutils.GetExtSrcNetworkEntityInfo("7", "7", "36.188.144.0/16", true)
	e8 := testutils.GetExtSrcNetworkEntityInfo("8", "8", "36.188.144.0/32", true)

	tree, err := NewNRadixTree(pkgNet.IPv4, []*storage.NetworkEntityInfo{e1, e2, e3, e4, e5, e6, e7, e8})
	assert.NoError(t, err)
	assert.NotNil(t, tree)

	assert.Equal(t, e1, tree.Get("1"))
	assert.Equal(t, e2, tree.Get("2"))
	assert.Equal(t, e3, tree.Get("3"))
	assert.Equal(t, e4, tree.Get("4"))
	assert.Equal(t, e5, tree.Get("5"))
	assert.Equal(t, e6, tree.Get("6"))
	assert.Equal(t, e7, tree.Get("7"))
	assert.Equal(t, e8, tree.Get("8"))

	assert.Error(t, tree.Insert(testutils.GetExtSrcNetworkEntityInfo("60", "60", "36.188.144.0/16", true)))

	assert.Equal(t, e2, tree.GetSupernet(e1.GetId()))
	assert.Equal(t, e1, tree.GetSupernet(e4.GetId()))
	assert.Equal(t, e7, tree.GetSupernet(e6.GetId()))

	assert.Equal(t, e2, tree.GetMatchingSupernet(e4.GetId(), func(e *storage.NetworkEntityInfo) bool {
		return !e.GetExternalSource().GetDefault()
	}))
	assert.Equal(t, e1, tree.GetMatchingSupernet(e4.GetId(), func(e *storage.NetworkEntityInfo) bool {
		return e.GetExternalSource().GetDefault()
	}))

	assert.Nil(t, tree.GetSupernetForCIDR("0.0.0.0/0"))
	assert.Equal(t, e2, tree.GetSupernetForCIDR("35.187.144.0/20"))
	assert.Equal(t, e2, tree.GetSupernetForCIDR("35.187.144.0/18"))

	assert.Equal(t, e3, tree.GetMatchingSupernetForCIDR("35.187.144.0/18", func(e *storage.NetworkEntityInfo) bool {
		return e.GetId() != e2.GetId()
	}))
	assert.Equal(t, e2, tree.GetMatchingSupernetForCIDR("35.187.144.0/18", nil))

	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e2, e5}, tree.GetSubnets(e3.GetId()))

	tree.Remove(e3.GetId())
	assert.Nil(t, tree.Get(e3.GetId()))

	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e2, e5}, tree.GetSubnetsForCIDR(e3.GetExternalSource().GetCidr()))
	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e2, e5, e7}, tree.GetSubnetsForCIDR("0.0.0.0/0"))
}

func TestNRadixTreeIPv6(t *testing.T) {
	e1 := testutils.GetExtSrcNetworkEntityInfo("1", "1", "2001:db8:3333:4444:5555:6666:7777:8888/63", true)
	e2 := testutils.GetExtSrcNetworkEntityInfo("2", "2", "2001:db8:3333:4444:5555:6666:7777:8888/64", false)
	e3 := testutils.GetExtSrcNetworkEntityInfo("3", "3", "2001:db8:3333:4444:5555:6666:7777:8888/100", false)
	e4 := testutils.GetExtSrcNetworkEntityInfo("4", "4", "2001:db8:3333:4444:5555:6666:7777:8888/128", false)
	e5 := testutils.GetExtSrcNetworkEntityInfo("5", "5", "2001:db8:2222:4444:5555:6666:7777:8888/70", true)
	e6 := testutils.GetExtSrcNetworkEntityInfo("6", "6", "2001:db8:2222:4444:5555:6666:7777:8888/80", false)

	tree, err := NewNRadixTree(pkgNet.IPv6, []*storage.NetworkEntityInfo{e1, e2, e3, e4, e5, e6})
	assert.NoError(t, err)
	assert.NotNil(t, tree)

	assert.Equal(t, e1, tree.Get("1"))
	assert.Equal(t, e2, tree.Get("2"))
	assert.Equal(t, e3, tree.Get("3"))
	assert.Equal(t, e4, tree.Get("4"))
	assert.Equal(t, e5, tree.Get("5"))
	assert.Equal(t, e6, tree.Get("6"))

	assert.Error(t, tree.Insert(testutils.GetExtSrcNetworkEntityInfo("60", "60", "2001:db8:2222:4444:5555:6666:7777:8888/80", true)))

	assert.Equal(t, e1, tree.GetSupernet(e2.GetId()))
	assert.Equal(t, networkgraph.InternetEntity().ToProto(), tree.GetSupernet(e1.GetId()))
	assert.Equal(t, e5, tree.GetSupernet(e6.GetId()))

	assert.Equal(t, e3, tree.GetMatchingSupernet(e4.GetId(), func(e *storage.NetworkEntityInfo) bool {
		return !e.GetExternalSource().GetDefault()
	}))
	assert.Equal(t, e1, tree.GetMatchingSupernet(e4.GetId(), func(e *storage.NetworkEntityInfo) bool {
		return e.GetExternalSource().GetDefault()
	}))

	assert.Nil(t, tree.GetSupernetForCIDR("::ffff:0:0/0"))
	assert.Equal(t, e2, tree.GetSupernetForCIDR("2001:db8:3333:4444:5555:6666:7777:8888/100"))
	assert.Equal(t, e2, tree.GetSupernetForCIDR("2001:db8:3333:4444:5555:6666:7777:8888/90"))

	assert.Equal(t, e1, tree.GetMatchingSupernetForCIDR("2001:db8:3333:4444:5555:6666:7777:8888/90", func(e *storage.NetworkEntityInfo) bool {
		return e.GetId() != e2.GetId()
	}))
	assert.Equal(t, e2, tree.GetMatchingSupernetForCIDR("2001:db8:3333:4444:5555:6666:7777:8888/90", nil))

	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e4}, tree.GetSubnets(e3.GetId()))

	tree.Remove(e3.GetId())
	assert.Nil(t, tree.Get(e3.GetId()))

	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e4}, tree.GetSubnetsForCIDR(e3.GetExternalSource().GetCidr()))
	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e1, e5}, tree.GetSubnetsForCIDR("::ffff:0:0/0"))
}
