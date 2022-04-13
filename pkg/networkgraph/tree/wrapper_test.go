package tree

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/networkgraph"
	"github.com/stackrox/stackrox/pkg/networkgraph/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNetworkTreeWrapper(t *testing.T) {
	/*

		ipv4:
			INTERNET
			 	|______ 3
						|__ 2
							|__ (1)
								 |__ 4
		ipv6:
			INTERNET
				|_____ (6)
						|__ 5

	*/

	e1 := testutils.GetExtSrcNetworkEntityInfo("1", "1", "35.187.144.0/20", true)
	e2 := testutils.GetExtSrcNetworkEntityInfo("2", "2", "35.187.144.0/16", false)
	e3 := testutils.GetExtSrcNetworkEntityInfo("3", "3", "35.187.144.0/8", false)
	e4 := testutils.GetExtSrcNetworkEntityInfo("4", "4", "35.187.144.0/23", false)
	e5 := testutils.GetExtSrcNetworkEntityInfo("5", "5", "::23:ffff:24bc:9000/126", false)
	e6 := testutils.GetExtSrcNetworkEntityInfo("6", "6", "::23:ffff:24bc:9000/112", true)

	treeWrapper, err := NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{e1, e2, e3, e4, e5, e6})
	assert.NoError(t, err)

	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e4}, treeWrapper.GetSubnets("1"))
	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e2}, treeWrapper.GetSubnets("3"))
	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{}, treeWrapper.GetSubnets("4"))
	assert.Equal(t, e5, treeWrapper.Get("5"))
	assert.Equal(t, e2, treeWrapper.GetSupernet("1"))
	assert.Equal(t, e6, treeWrapper.GetSupernet("5"))
	assert.Equal(t, networkgraph.InternetEntity().ToProto(), treeWrapper.GetSupernet(networkgraph.InternetExternalSourceID))

	e7 := testutils.GetExtSrcNetworkEntityInfo("7", "7", "::23:ffff:24bc:9000/127", false)
	e8 := testutils.GetExtSrcNetworkEntityInfo("8", "8", "35.188.144.0/5", false)

	assert.NoError(t, treeWrapper.Insert(e7))
	assert.NoError(t, treeWrapper.Insert(e8))

	/*
		Expected trees after inserts:

		ipv4:
			INTERNET
			 	|______ 8*
						|___3
							|__ 2
								|__ (1)
									 |__ 4
		ipv6:
			INTERNET
				|_____ (6)
						|__ 5
							|__ 7*

	*/

	assert.Equal(t, e5, treeWrapper.GetSupernet("7"))
	assert.Equal(t, e8, treeWrapper.GetSupernet("3"))
	assert.Equal(t, networkgraph.InternetEntity().ToProto(), treeWrapper.GetSupernet("6"))
	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e6, e8}, treeWrapper.GetSubnets(networkgraph.InternetExternalSourceID))

	// Invalid CIDR
	assert.Error(t, treeWrapper.Insert(testutils.GetExtSrcNetworkEntityInfo("9", "9", "0::0:0:0:0:0:ffff:0:0/0", false)))

	// Update CIDR
	e4 = testutils.GetExtSrcNetworkEntityInfo("4", "4", "35.187.144.0/26", false)
	assert.NoError(t, treeWrapper.Insert(e4))
	assert.Equal(t, e4, treeWrapper.Get("4"))

	// Existing CIDR
	assert.Error(t, treeWrapper.Insert(testutils.GetExtSrcNetworkEntityInfo("88", "88", "35.188.144.0/5", false)))

	assert.Equal(t, e2, treeWrapper.GetMatchingSupernet("4", func(e *storage.NetworkEntityInfo) bool { return !e.GetExternalSource().GetDefault() }))
	assert.Equal(t, e1, treeWrapper.GetMatchingSupernet("4", func(e *storage.NetworkEntityInfo) bool { return e.GetExternalSource().GetDefault() }))

	treeWrapper.Remove("1")
	assert.Nil(t, treeWrapper.Get("1"))

	/*
		Expected trees after remove:

		ipv4:
			INTERNET
			 	|______ 8*
						|___3
							|__ 2
								|__ (1)
									 |__ 4
		ipv6:
			INTERNET
				|_____ (6)
						|__ 5
							|__ 7*

	*/
	assert.Equal(t, e2, treeWrapper.GetMatchingSupernet("4", func(e *storage.NetworkEntityInfo) bool { return !e.GetExternalSource().GetDefault() }))

	// Existing entity different IP address family
	e8 = testutils.GetExtSrcNetworkEntityInfo("8", "8", "::23:ffff:24bc:9000/100", false)
	assert.NoError(t, treeWrapper.Insert(e8))

	/*
		Expected trees after insert:

		ipv4:
			INTERNET
			 	|
				|___________3
							|__ 2
								|__ (1)
									 |__ 4
		ipv6:
			INTERNET
				|_____ 8
					   |__(6)
						   |__ 5
							   |__ 7*

	*/
	assert.Equal(t, e8, treeWrapper.Get("8"))
	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e6}, treeWrapper.GetSubnets("8"))
	assert.Equal(t, networkgraph.InternetEntity().ToProto(), treeWrapper.GetSupernet("8"))

	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e3}, treeWrapper.GetSubnetsForCIDR("35.0.0.0/6"))

	assert.Equal(t, e3, treeWrapper.GetSupernetForCIDR("35.187.144.0/14"))
}
