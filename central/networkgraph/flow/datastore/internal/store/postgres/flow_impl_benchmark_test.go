//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/require"
)

func BenchmarkGetAllFlows(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(context.Background(), fixtureconsts.Cluster1)
	require.NoError(b, err)

	// 25000 flows
	setupExternalIngressFlows(b, flowStore, fixtureconsts.Deployment1, 1000)
	setupExternalIngressFlows(b, flowStore, fixtureconsts.Deployment2, 10000)

	setupExternalIngressFlows(b, flowStore, fixtureconsts.Deployment3, 1000)
	setupExternalEgressFlows(b, flowStore, fixtureconsts.Deployment3, 1000)

	setupDeploymentFlows(b, flowStore, fixtureconsts.Deployment1, fixtureconsts.Deployment2, 1000)
	setupDeploymentFlows(b, flowStore, fixtureconsts.Deployment2, fixtureconsts.Deployment4, 1000)
	setupDeploymentFlows(b, flowStore, fixtureconsts.Deployment3, fixtureconsts.Deployment4, 10000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		since := time.Now().Add(-5 * time.Minute)
		_, _, err := flowStore.GetAllFlows(context.Background(), &since)
		require.NoError(b, err)
	}
}

func BenchmarkUpsertFlows(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(context.Background(), fixtureconsts.Cluster1)
	require.NoError(b, err)

	// These benchmarks are relevant individually, but must be carefully compared.
	// for the single flow insertion, we're only ever inserting 1 * b.N flows,
	// for the batch insertions, it's 100 * b.N and 1000 * b.N respectively, so results
	// must be compared with consideration for the total number of flows upserted into the database
	b.Run("benchmark upsert single flow", benchmarkUpsertFlows(flowStore, 1))
	b.Run("benchmark upsert 100 flow batch", benchmarkUpsertFlows(flowStore, 100))
	b.Run("benchmark upsert 1000 flow batch", benchmarkUpsertFlows(flowStore, 1000))
}

func benchmarkUpsertFlows(flowStore store.FlowStore, numFlows uint32) func(*testing.B) {
	return func(b *testing.B) {
		flows := make([]*storage.NetworkFlow, 0, numFlows)
		for i := uint32(0); i < numFlows; i++ {
			id, err := testutils.ExtIdFromIPv4(fixtureconsts.Cluster1, i)
			require.NoError(b, err)
			flows = append(flows, testutils.ExtFlow(fixtureconsts.Deployment1, id.String()))
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := flowStore.UpsertFlows(context.Background(), flows, timestamp.Now()-1000000)
			require.NoError(b, err)

			exclude(b, func() {
				err = flowStore.RemoveFlowsForDeployment(context.Background(), fixtureconsts.Deployment1)
				require.NoError(b, err)
			})
		}
	}
}

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
		flows = append(flows, testutils.DepFlow(toId, fromId))
	}

	err := flowStore.UpsertFlows(context.Background(), flows, timestamp.Now()-1000000)
	require.NoError(b, err)
}

func setupExternalIngressFlows(b *testing.B, flowStore store.FlowStore, deploymentId string, numFlows uint32) {
	flows := make([]*storage.NetworkFlow, 0, numFlows)
	for i := uint32(0); i < numFlows; i++ {
		id, err := testutils.ExtIdFromIPv4(fixtureconsts.Cluster1, i)
		require.NoError(b, err)
		flows = append(flows, testutils.ExtFlow(deploymentId, id.String()))
	}

	err := flowStore.UpsertFlows(context.Background(), flows, timestamp.Now()-1000000)
	require.NoError(b, err)
}

func setupExternalEgressFlows(b *testing.B, flowStore store.FlowStore, deploymentId string, numFlows uint32) {
	flows := make([]*storage.NetworkFlow, 0, numFlows)
	for i := uint32(0); i < numFlows; i++ {
		id, err := testutils.ExtIdFromIPv4(fixtureconsts.Cluster1, i)
		require.NoError(b, err)

		flows = append(flows, testutils.ExtFlow(id.String(), deploymentId))
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

func exclude(b *testing.B, f func()) {
	b.StopTimer()
	f()
	b.StartTimer()
}
