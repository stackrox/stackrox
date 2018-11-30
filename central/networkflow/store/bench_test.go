package store

import (
	"fmt"
	"math"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkentity"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

var log = logging.LoggerForModule()

func getFlows(maxNetworkFlows int) []*v1.NetworkFlow {
	numDeployments := int(math.Sqrt(float64(maxNetworkFlows)))

	flows := make([]*v1.NetworkFlow, 0, numDeployments*numDeployments)
	for i := 0; i < numDeployments; i++ {
		for j := 0; j < numDeployments; j++ {
			flow := &v1.NetworkFlow{
				Props: &v1.NetworkFlowProperties{
					SrcEntity:  networkentity.ForDeployment(fmt.Sprintf("%d", i)).ToProto(),
					DstEntity:  networkentity.ForDeployment(fmt.Sprintf("%d", j)).ToProto(),
					L4Protocol: v1.L4Protocol_L4_PROTOCOL_TCP,
					DstPort:    80,
				},
			}
			flows = append(flows, flow)
		}
	}
	return flows
}

func preloadDB(t require.TestingT, preload int) (int, FlowStore) {
	boltdb, err := bolthelper.NewTemp("bench_test.db")
	require.NoError(t, err)

	clusterStore := NewClusterStore(boltdb)
	clusterStore.CreateFlowStore("cluster1")
	flowStore := clusterStore.GetFlowStore("cluster1")

	preloadFlows := getFlows(preload)
	err = flowStore.UpsertFlows(preloadFlows, timestamp.Now())
	require.NoError(t, err)
	return len(preloadFlows), flowStore
}

func benchmarkUpdate(b *testing.B, preload, postload int) {
	_, flowStore := preloadDB(b, preload)
	postloadFlows := getFlows(postload)
	for i := 0; i < b.N; i++ {
		flowStore.UpsertFlows(postloadFlows, timestamp.Now())
	}
}

func BenchmarkLeveledFlows(b *testing.B) {
	var cases = []struct {
		preload  int
		postload int
	}{
		{
			preload:  10000,
			postload: 4,
		},
		{
			preload:  25000,
			postload: 4,
		},
		{
			preload:  100000,
			postload: 4,
		},
	}
	for _, c := range cases {
		b.Run(fmt.Sprintf("%d-%d", c.preload, c.postload), func(b *testing.B) {
			benchmarkUpdate(b, c.preload, c.postload)
		})
	}
}

func BenchmarkGetID(b *testing.B) {
	props := &v1.NetworkFlowProperties{
		SrcEntity:  networkentity.ForDeployment(uuid.NewV4().String()).ToProto(),
		DstEntity:  networkentity.ForDeployment(uuid.NewV4().String()).ToProto(),
		DstPort:    9999,
		L4Protocol: v1.L4Protocol_L4_PROTOCOL_UDP,
	}
	for i := 0; i < b.N; i++ {
		getID(props)
	}
}
