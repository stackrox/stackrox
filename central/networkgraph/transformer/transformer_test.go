package transformer

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
)

func TestTransformDiscoveredEntities(t *testing.T) {
	d1 := testutils.GetDeploymentNetworkEntity("d1", "d1")
	d2 := testutils.GetDeploymentNetworkEntity("d2", "d2")

	entities := []*storage.NetworkEntityInfo{
		testutils.GetExtSrcNetworkEntityInfo("entity1", "1", "1.1.1.1/32", false, true),
		testutils.GetExtSrcNetworkEntityInfo("entity2", "2", "2.2.2.2/32", false, false),
	}

	ts := time.Now()

	flows := []*storage.NetworkFlow{
		testutils.GetNetworkFlow(d1, entities[0], 8000, storage.L4Protocol_L4_PROTOCOL_TCP, &ts),
		testutils.GetNetworkFlow(d1, entities[1], 8000, storage.L4Protocol_L4_PROTOCOL_TCP, &ts),
		testutils.GetNetworkFlow(entities[1], d2, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, &ts),
		testutils.GetNetworkFlow(d1, d2, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, &ts),
	}

	expected := []*storage.NetworkFlow{
		testutils.GetNetworkFlow(d1, networkgraph.InternetEntity().ToProto(), 8000, storage.L4Protocol_L4_PROTOCOL_TCP, &ts),
		testutils.GetNetworkFlow(d1, entities[1], 8000, storage.L4Protocol_L4_PROTOCOL_TCP, &ts),
		testutils.GetNetworkFlow(entities[1], d2, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, &ts),
		testutils.GetNetworkFlow(d1, d2, 8000, storage.L4Protocol_L4_PROTOCOL_TCP, &ts),
	}

	result := NewExternalDiscoveredTransformer().Transform(flows)

	protoassert.ElementsMatch(t, expected, result)
}
