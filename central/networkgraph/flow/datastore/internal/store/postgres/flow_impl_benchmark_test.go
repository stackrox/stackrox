//go:build sql_integration

package postgres

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/netip"
	"testing"

	// "time"

	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/postgres/pgtest"

	// "github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/require"
)

func BenchmarkGetExternalFlows(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(context.Background(), fixtureconsts.Cluster1)
	require.NoError(b, err)

	setupExternalIngressFlows(b, flowStore, fixtureconsts.Deployment1, 1000)
	setupExternalIngressFlows(b, flowStore, fixtureconsts.Deployment2, 10000)

	setupExternalIngressFlows(b, flowStore, fixtureconsts.Deployment3, 1000)
	setupExternalEgressFlows(b, flowStore, fixtureconsts.Deployment3, 1000)

	b.Run("deployment with 1000 external flows", benchmarkGetExternalFlows(flowStore, fixtureconsts.Deployment1))
	b.Run("deployment with 10000 external flows", benchmarkGetExternalFlows(flowStore, fixtureconsts.Deployment2))
	b.Run("deployment with 1000 ingress and 1000 egress flows", benchmarkGetExternalFlows(flowStore, fixtureconsts.Deployment3))
}

func BenchmarkGetFlowsForDeployment(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(context.Background(), fixtureconsts.Cluster1)
	require.NoError(b, err)

	// 1000 flows 1 -> 2
	setupDeploymentFlows(b, flowStore, fixtureconsts.Deployment1, fixtureconsts.Deployment2, 1000)
	// 1000 flows 2 -> 4 (2 == 2000 flows total)
	setupDeploymentFlows(b, flowStore, fixtureconsts.Deployment2, fixtureconsts.Deployment4, 1000)
	// 10000 flows 3 -> 4 (4 == 11000 flows total)
	setupDeploymentFlows(b, flowStore, fixtureconsts.Deployment3, fixtureconsts.Deployment4, 10000)

	b.Run("deployment with 1000 flows", benchmarkGetFlowsForDeployment(flowStore, fixtureconsts.Deployment1))
	b.Run("deployment with 10000 flows", benchmarkGetFlowsForDeployment(flowStore, fixtureconsts.Deployment3))
	b.Run("deployment with 1000 ingress and 1000 egress flows", benchmarkGetFlowsForDeployment(flowStore, fixtureconsts.Deployment2))
}

func setupDeploymentFlows(b *testing.B, flowStore store.FlowStore, fromId string, toId string, numFlows int) {
	flows := make([]*storage.NetworkFlow, 0, numFlows)
	for i := 0; i < numFlows; i++ {
		flows = append(flows, anyFlow(toId, storage.NetworkEntityInfo_DEPLOYMENT, fromId, storage.NetworkEntityInfo_DEPLOYMENT))
	}

	err := flowStore.UpsertFlows(context.Background(), flows, timestamp.Now()-1000000)
	require.NoError(b, err)
}

func setupExternalIngressFlows(b *testing.B, flowStore store.FlowStore, deploymentId string, numFlows int) {
	flows := make([]*storage.NetworkFlow, 0, numFlows)
	for i := 0; i < numFlows; i++ {
		flows = append(flows, extFlow(deploymentId, extId(fixtureconsts.Cluster1, i)))
	}

	err := flowStore.UpsertFlows(context.Background(), flows, timestamp.Now()-1000000)
	require.NoError(b, err)
}

func setupExternalEgressFlows(b *testing.B, flowStore store.FlowStore, deploymentId string, numFlows int) {
	flows := make([]*storage.NetworkFlow, 0, numFlows)
	for i := 0; i < numFlows; i++ {
		flows = append(flows, extFlow(extId(fixtureconsts.Cluster1, i), deploymentId))
	}

	err := flowStore.UpsertFlows(context.Background(), flows, timestamp.Now()-1000000)
	require.NoError(b, err)
}

func benchmarkGetExternalFlows(flowStore store.FlowStore, deploymentId string) func(*testing.B) {
	return func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := flowStore.GetExternalFlowsForDeployment(context.Background(), deploymentId)
			require.NoError(b, err)
		}
	}
}

func benchmarkGetFlowsForDeployment(flowStore store.FlowStore, deploymentId string) func(*testing.B) {
	return func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := flowStore.GetFlowsForDeployment(context.Background(), deploymentId)
			require.NoError(b, err)
		}
	}
}

func anyFlow(toID string, toType storage.NetworkEntityInfo_Type, fromID string, fromType storage.NetworkEntityInfo_Type) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: fromType,
				Id:   fromID,
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: toType,
				Id:   toID,
			},
		},
	}
}

func extFlow(toID, fromID string) *storage.NetworkFlow {
	return anyFlow(toID, storage.NetworkEntityInfo_EXTERNAL_SOURCE, fromID, storage.NetworkEntityInfo_DEPLOYMENT)
}

func depFlow(toID, fromID string) *storage.NetworkFlow {
	return anyFlow(toID, storage.NetworkEntityInfo_DEPLOYMENT, fromID, storage.NetworkEntityInfo_DEPLOYMENT)
}

func extId(clusterId string, idx int) string {
	bs := [4]byte{}
	binary.BigEndian.PutUint32(bs[:], uint32(idx))
	ip := netip.AddrFrom4(bs)
	resource, _ := externalsrcs.NewClusterScopedID(clusterId, fmt.Sprintf("%s/32", ip.String()))
	return resource.String()
}
