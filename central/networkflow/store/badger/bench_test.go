package badger

import (
	"fmt"
	"math"
	"testing"

	"github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkentity"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

var (
	log = logging.LoggerForModule()
)

func getFlows(maxNetworkFlows int) []*storage.NetworkFlow {
	numDeployments := int(math.Sqrt(float64(maxNetworkFlows)))

	flows := make([]*storage.NetworkFlow, 0, numDeployments*numDeployments)
	for i := 0; i < numDeployments; i++ {
		for j := 0; j < numDeployments; j++ {
			flow := &storage.NetworkFlow{
				Props: &storage.NetworkFlowProperties{
					SrcEntity:  networkentity.ForDeployment(fmt.Sprintf("%d", i)).ToProto(),
					DstEntity:  networkentity.ForDeployment(fmt.Sprintf("%d", j)).ToProto(),
					L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					DstPort:    80,
				},
			}
			flows = append(flows, flow)
		}
	}
	return flows
}

func preloadDB(t require.TestingT, preload int) (int, store.FlowStore) {
	db, _, err := badgerhelper.NewTemp("")
	require.NoError(t, err)

	clusterStore := NewClusterStore(db)
	_, err = clusterStore.CreateFlowStore("cluster1")
	require.NoError(t, err)
	flowStore := clusterStore.GetFlowStore("cluster1")

	preloadFlows := getFlows(preload)
	b := batcher.New(len(preloadFlows), 500)

	for {
		start, end, valid := b.Next()
		if !valid {
			break
		}
		err = flowStore.UpsertFlows(preloadFlows[start:end], timestamp.Now())
		require.NoError(t, err)
	}
	return len(preloadFlows), flowStore
}

func benchmarkUpdate(b *testing.B, preload, postload int) {
	_, flowStore := preloadDB(b, preload)
	postloadFlows := getFlows(postload)
	for i := 0; i < b.N; i++ {
		require.NoError(b, flowStore.UpsertFlows(postloadFlows, timestamp.Now()))
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
	props := &storage.NetworkFlowProperties{
		SrcEntity:  networkentity.ForDeployment(uuid.NewV4().String()).ToProto(),
		DstEntity:  networkentity.ForDeployment(uuid.NewV4().String()).ToProto(),
		DstPort:    9999,
		L4Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
	}
	s := flowStoreImpl{keyPrefix: uuid.NewV4().Bytes()}
	for i := 0; i < b.N; i++ {
		s.getID(props)
	}
}
