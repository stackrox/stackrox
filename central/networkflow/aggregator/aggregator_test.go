package aggregator

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/test"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stretchr/testify/assert"
)

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
	e1 := test.GetExtSrcNetworkEntity("1", "1", "35.187.144.0/20", true)
	e2 := test.GetExtSrcNetworkEntity("2", "2", "35.187.144.0/16", false)
	e3 := test.GetExtSrcNetworkEntity("3", "3", "35.187.144.0/8", false)
	e4 := test.GetExtSrcNetworkEntity("4", "4", "35.187.144.0/23", true)
	e5 := test.GetExtSrcNetworkEntity("5", "5", "36.188.144.0/30", false)
	e6 := test.GetExtSrcNetworkEntity("6", "6", "36.188.144.0/16", true)

	networkTree, err := tree.NewNetworkTreeWrapper([]*storage.NetworkEntityInfo{e1, e2, e3, e4, e5, e6})
	assert.NoError(t, err)

	assert.Equal(t, e2, networkTree.GetSupernet("1"))
	assert.Equal(t, internet, networkTree.GetSupernet("6"))
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

	actual := NewDefaultToCustomExtSrcAggregator(networkTree).Aggregate(flows)
	assert.ElementsMatch(t, expected, actual)
}
