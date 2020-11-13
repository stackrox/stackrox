package aggregator

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/networkgraph/test"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stretchr/testify/assert"
)

func TestSubnetToSupernetAggregator(t *testing.T) {
	d1 := test.GetDeploymentNetworkEntity("d1", "d1")
	d2 := test.GetDeploymentNetworkEntity("d2", "d2")

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


	*/

	internet := networkgraph.InternetEntity().ToProto()

	cidr1 := "35.187.144.0/20"
	cidr2 := "35.187.144.0/16"
	cidr3 := "35.187.144.0/8"
	cidr4 := "35.187.144.0/23"
	cidr5 := "36.188.144.0/30"
	cidr6 := "36.188.144.0/16"

	id1, _ := externalsrcs.NewClusterScopedID("1", cidr1)
	id2, _ := externalsrcs.NewClusterScopedID("1", cidr2)
	id3, _ := externalsrcs.NewClusterScopedID("1", cidr3)
	id4, _ := externalsrcs.NewClusterScopedID("1", cidr4)
	id5, _ := externalsrcs.NewClusterScopedID("1", cidr5)
	id6, _ := externalsrcs.NewClusterScopedID("1", cidr6)

	e1 := test.GetExtSrcNetworkEntityInfo(id1.String(), "1", cidr1, true)  // -> e2
	e2 := test.GetExtSrcNetworkEntityInfo(id2.String(), "2", cidr2, false) // -> e3
	e3 := test.GetExtSrcNetworkEntityInfo(id3.String(), "3", cidr3, false) // -> internet
	e4 := test.GetExtSrcNetworkEntityInfo(id4.String(), "4", cidr4, true)  // -> e1
	e5 := test.GetExtSrcNetworkEntityInfo(id5.String(), "5", cidr5, false) // -> e6
	e6 := test.GetExtSrcNetworkEntityInfo(id6.String(), "6", cidr6, true)  // -> internet

	tree1, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{e2, e3, e5})
	assert.NoError(t, err)
	tree2, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{e1, e4, e6})
	assert.NoError(t, err)

	/*

		flows:

			f1: d1 -> e1:8000/tcp:ts1
			f2: d1 -> e2:8000/tcp:ts2
			f3: d1 -> e5:8000/tcp
			f4: d1 -> e6
			f5: e6 -> d2
			f6: e6 -> d2:8000:ts2
			f7: internet -> d2
			f8: internet -> d2:8000:ts1
			f9: d2 -> e4:8000/tcp:ts2
			f10: d2 -> e4:8000/tcp:ts2
	*/

	ts1 := types.TimestampNow()
	ts2 := ts1.Clone()
	ts2.Seconds = ts2.Seconds + 1000

	f1 := test.GetNetworkFlow(d1, e1, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts1)
	f2 := test.GetNetworkFlow(d1, e2, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)
	f3 := test.GetNetworkFlow(d1, e5, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, nil)
	f4 := test.GetNetworkFlow(d1, e6, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f5 := test.GetNetworkFlow(e6, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f6 := test.GetNetworkFlow(e6, d2, 8000, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, ts2)
	f7 := test.GetNetworkFlow(internet, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f8 := test.GetNetworkFlow(internet, d2, 8000, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, ts1)
	f9 := test.GetNetworkFlow(d1, e4, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)
	f10 := test.GetNetworkFlow(d2, e4, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)

	flows := []*storage.NetworkFlow{f1, f2, f3, f4, f5, f6, f7, f8, f9, f10}

	f1x := test.GetNetworkFlow(d1, e2, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts1)
	f2x := test.GetNetworkFlow(d1, e3, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)
	f3x := test.GetNetworkFlow(d1, e6, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, nil)
	f4x := test.GetNetworkFlow(d1, internet, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f5x := test.GetNetworkFlow(internet, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f6x := test.GetNetworkFlow(internet, d2, 8000, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, ts2)
	f9x := test.GetNetworkFlow(d1, e1, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)
	f10x := test.GetNetworkFlow(d2, e1, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)

	expected := []*storage.NetworkFlow{f1x, f2x, f3x, f4x, f5x, f6x, f9x, f10x}

	aggr, err := NewSubnetToSupernetConnAggregator(tree.NewMultiTreeWrapper(tree1, tree2))
	assert.NoError(t, err)
	actual := aggr.Aggregate(flows)
	assert.ElementsMatch(t, expected, actual)
}

func TestHideDefaultExtSrcsAggregator(t *testing.T) {
	d1 := test.GetDeploymentNetworkEntity("d1", "d1")
	d2 := test.GetDeploymentNetworkEntity("d2", "d2")

	/*

		Network tree from test networks:

			INTERNET
			 	|______ 3
				|		|__ 2
				|			|__ (1)
				|				 |__ (4)
				|______ (6)
						|__ 5


		Network tree on without default networks. Note that this just to help understand where the connections are
		mapped when default networks are hidden. The actual network tree remains unchanged.

			INTERNET` (INTERNET + 6)
			 	|______ 3
				|		|__ 2' (2 + 1 + 4)
				|
				|
				|______ 5


	*/

	internet := networkgraph.InternetEntity().ToProto()
	e1 := test.GetExtSrcNetworkEntityInfo("1", "1", "35.187.144.0/20", true)
	e2 := test.GetExtSrcNetworkEntityInfo("2", "2", "35.187.144.0/16", false)
	e3 := test.GetExtSrcNetworkEntityInfo("3", "3", "35.187.144.0/8", false)
	e4 := test.GetExtSrcNetworkEntityInfo("4", "4", "35.187.144.0/23", true)
	e5 := test.GetExtSrcNetworkEntityInfo("5", "5", "36.188.144.0/30", false)
	e6 := test.GetExtSrcNetworkEntityInfo("6", "6", "36.188.144.0/16", true)

	networkTree, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{e1, e2, e3, e4, e5, e6})
	assert.NoError(t, err)

	assert.Equal(t, e2, networkTree.GetSupernet("1"))
	assert.Equal(t, internet.GetId(), networkTree.GetSupernet("6").GetId())
	/*

		flows without hiding default networks:

			f1: d1 -> e1*:8000/tcp
			f2: d1 -> e2:8000/tcp
			f3: d1 -> e5
			f4: d1 -> e6*
			f5: e6* -> d2
			f6: e6* -> d2
			f7: internet -> d2
			f8: internet -> d2:8000
			f9: d2 -> e4*

		flows after hiding default networks:

			f2:  d1 -> e2:8000/tcp
			f3:  d1 -> e5
			f4x: d1 -> internet
			f7:  internet -> d2
			f8x: internet -> d2:8000 (ts updated)

		movement
		f1 --> f2
		       f2
			   f3
		f4 --> f4x
			   f5
		f6 --> f8x
			   f7
		f8 --> f8x
		f9 --> f2
		f10--> f10x

	*/

	ts1 := types.TimestampNow()
	ts2 := ts1.Clone()
	ts2.Seconds = ts2.Seconds + 1000

	f1 := test.GetNetworkFlow(d1, e1, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts1)
	f2 := test.GetNetworkFlow(d1, e2, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)
	f3 := test.GetNetworkFlow(d1, e5, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, nil)
	f4 := test.GetNetworkFlow(d1, e6, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f5 := test.GetNetworkFlow(e6, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f6 := test.GetNetworkFlow(e6, d2, 8000, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, ts2)
	f7 := test.GetNetworkFlow(internet, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f8 := test.GetNetworkFlow(internet, d2, 8000, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, ts1)
	f9 := test.GetNetworkFlow(d1, e4, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)
	f10 := test.GetNetworkFlow(d2, e4, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)

	flows := []*storage.NetworkFlow{f1, f2, f3, f4, f5, f6, f7, f8, f9, f10}

	f4x := test.GetNetworkFlow(d1, internet, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f8x := test.GetNetworkFlow(internet, d2, 8000, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, ts2)
	f10x := test.GetNetworkFlow(d2, e2, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)

	expected := []*storage.NetworkFlow{f2, f3, f4x, f7, f8x, f10x}

	aggr, err := NewDefaultToCustomExtSrcConnAggregator(networkTree)
	assert.NoError(t, err)
	actual := aggr.Aggregate(flows)
	assert.ElementsMatch(t, expected, actual)
}

func TestAggregateExtConnsByName(t *testing.T) {
	ts1 := types.TimestampNow()
	ts2 := ts1.Clone()
	ts2.Seconds = ts2.Seconds + 1000

	d1 := test.GetDeploymentNetworkEntity("d1", "d1")
	d2 := test.GetDeploymentNetworkEntity("d2", "d2")

	e1 := test.GetExtSrcNetworkEntityInfo("cluster1__e1", "google", "net1", false)
	e2 := test.GetExtSrcNetworkEntityInfo("cluster1__e2", "google", "net2", false)
	e3 := test.GetExtSrcNetworkEntityInfo("cluster1__e3", "google", "net3", false)
	e4 := test.GetExtSrcNetworkEntityInfo("cluster1__id4", "e4", "", false)
	e5 := test.GetExtSrcNetworkEntityInfo("cluster1__nameless", "", "", false)
	e6 := test.GetExtSrcNetworkEntityInfo("cluster1__e6", "extSrc6", "net6", false)

	f1 := test.GetNetworkFlow(d1, e1, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts1)
	f2 := test.GetNetworkFlow(d1, e2, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)
	f3 := test.GetNetworkFlow(d1, e3, 8080, storage.L4Protocol_L4_PROTOCOL_TCP, nil)
	f4 := test.GetNetworkFlow(d1, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f5 := test.GetNetworkFlow(e4, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f6 := test.GetNetworkFlow(e5, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f7 := f6
	f8 := test.GetNetworkFlow(e5, d2, 8080, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f9 := test.GetNetworkFlow(e6, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f10 := test.GetNetworkFlow(d2, e6, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)

	flows := []*storage.NetworkFlow{f1, f2, f3, f4, f5, f6, f7, f8, f9, f10}

	e2x := &storage.NetworkEntityInfo{
		Id:   "cluster1__google",
		Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
		Desc: &storage.NetworkEntityInfo_ExternalSource_{
			ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
				Name: "google",
			},
		},
	}
	e5x := test.GetExtSrcNetworkEntityInfo("cluster1__nameless", "unnamed external source #1", "", false)

	/*
		f1 -> f2x
		f2 -> f2x
		f3 -> f3x
			  f4
			  f5
		f6 -> f6x
		f7 -> f6x
		f8 -> f8x
	*/

	f2x := test.GetNetworkFlow(d1, e2x, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, ts2)
	f3x := test.GetNetworkFlow(d1, e2x, 8080, storage.L4Protocol_L4_PROTOCOL_TCP, nil)
	f6x := test.GetNetworkFlow(e5x, d2, 0, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)
	f8x := test.GetNetworkFlow(e5x, d2, 8080, storage.L4Protocol_L4_PROTOCOL_UNKNOWN, nil)

	expected := []*storage.NetworkFlow{f2x, f3x, f4, f5, f6x, f8x, f9, f10}

	actual := NewDuplicateNameExtSrcConnAggregator().Aggregate(flows)

	assert.Len(t, actual, len(expected))
	assert.ElementsMatch(t, expected, actual)
}
