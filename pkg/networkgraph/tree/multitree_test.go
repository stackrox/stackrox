package tree

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stretchr/testify/assert"
)

func TestMultiTree(t *testing.T) {
	/*

		Network tree from test networks:

		tree1:
			INTERNET
			 	|______ 3
				|		|__ 2
				|
				|
				|______ 5

		tree2:
			INTERNET
			 	|______ 1
				|		|__ 4
				|
				|
				|______ 6


		combined:
			INTERNET
				|______ 3
				|		|__ 2
				|			|__ 1
				|
				|______ 6
						|__ 5

	*/

	e1 := testutils.GetExtSrcNetworkEntityInfo("1", "1", "35.187.144.0/20", true)
	e2 := testutils.GetExtSrcNetworkEntityInfo("2", "2", "35.187.144.0/16", false)
	e3 := testutils.GetExtSrcNetworkEntityInfo("3", "3", "35.187.144.0/8", false)
	e4 := testutils.GetExtSrcNetworkEntityInfo("4", "4", "35.187.144.0/23", true)
	e5 := testutils.GetExtSrcNetworkEntityInfo("5", "5", "36.188.144.0/30", false)
	e6 := testutils.GetExtSrcNetworkEntityInfo("6", "6", "36.188.144.0/16", true)

	tree1, err := NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{e2, e3, e5})
	assert.NoError(t, err)
	tree2, err := NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{e1, e4, e6})
	assert.NoError(t, err)

	multiTree := NewMultiNetworkTree(tree1, tree2)

	assert.Equal(t, e1, multiTree.Get("1"))

	assert.Equal(t, e2, multiTree.GetSupernet("1"))
	assert.Equal(t, e6, multiTree.GetSupernet("5"))

	assert.Equal(t, networkgraph.InternetExternalSourceID, multiTree.GetMatchingSupernet("5", func(entity *storage.NetworkEntityInfo) bool {
		return entity.GetId() != e6.GetId()
	}).GetId())

	assert.Equal(t, e6, multiTree.GetSupernetForCIDR("36.188.144.0/24"))

	assert.Equal(t, networkgraph.InternetExternalSourceID, multiTree.GetMatchingSupernetForCIDR("36.188.144.0/24", func(entity *storage.NetworkEntityInfo) bool {
		return entity.GetId() != e6.GetId()
	}).GetId())

	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e3, e6}, multiTree.GetSubnets(networkgraph.InternetExternalSourceID))

	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e3, e6}, multiTree.GetSubnetsForCIDR("32.0.0.0/5"))
}
