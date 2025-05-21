//go:build sql_integration

package tests

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/netip"
	"testing"
	"time"

	"github.com/google/uuid"

	entityStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	postgresFlowStore "github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/require"
)

var (
	log          = logging.LoggerForModule()
	ctx          = context.Background()
	allAccessCtx = sac.WithAllAccess(ctx)
)

func BenchmarkGetAllFlows(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := postgresFlowStore.NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(ctx, fixtureconsts.Cluster1)
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
		_, _, err := flowStore.GetAllFlows(ctx, &since)
		require.NoError(b, err)
	}
}

func BenchmarkUpsertFlows(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := postgresFlowStore.NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(ctx, fixtureconsts.Cluster1)
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
			flows = append(flows, testutils.ExtFlowIngress(fixtureconsts.Deployment1, id.String(), clusterID))
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := flowStore.UpsertFlows(ctx, flows, timestamp.Now()-1000000)
			require.NoError(b, err)

			exclude(b, func() {
				err = flowStore.RemoveFlowsForDeployment(ctx, fixtureconsts.Deployment1)
				require.NoError(b, err)
			})
		}
	}
}

func BenchmarkGetExternalFlows(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := postgresFlowStore.NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(ctx, fixtureconsts.Cluster1)
	require.NoError(b, err)

	setupExternalIngressFlows(b, flowStore, fixtureconsts.Deployment1, 1000)
	setupExternalIngressFlows(b, flowStore, fixtureconsts.Deployment2, 10000)

	setupExternalIngressFlows(b, flowStore, fixtureconsts.Deployment3, 1000)
	setupExternalEgressFlows(b, flowStore, fixtureconsts.Deployment3, 1000)

	b.Run("deployment with 1000 external flows", benchmarkGetExternalFlows(flowStore, fixtureconsts.Deployment1))
	b.Run("deployment with 10000 external flows", benchmarkGetExternalFlows(flowStore, fixtureconsts.Deployment2))
	b.Run("deployment with 1000 ingress and 1000 egress flows", benchmarkGetExternalFlows(flowStore, fixtureconsts.Deployment3))
}

func BenchmarkPruneOrphanedFlowsForDeployment(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := postgresFlowStore.NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(ctx, fixtureconsts.Cluster1)
	require.NoError(b, err)
	eStore := entityStore.GetTestPostgresDataStore(b, psql)

	b.Run("1000 flows to be pruned", benchmarkPruneOrphanedFlowsForDeployment(flowStore, eStore, fixtureconsts.Deployment1, 1000))
	b.Run("10000 flows to be pruned", benchmarkPruneOrphanedFlowsForDeployment(flowStore, eStore, fixtureconsts.Deployment1, 10000))
	b.Run("100000 flows to be pruned", benchmarkPruneOrphanedFlowsForDeployment(flowStore, eStore, fixtureconsts.Deployment1, 100000))
}

func BenchmarkRemoveOrphanedFlows(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := postgresFlowStore.NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(ctx, fixtureconsts.Cluster1)
	require.NoError(b, err)
	eStore := entityStore.GetTestPostgresDataStore(b, psql)

	// 100 deployments and 10 entities
	b.Run("1000 flows and 10 entities to be pruned", benchmarkRemoveOrphanedFlows(flowStore, eStore, fixtureconsts.Deployment1, 100, 10))
	// 100 deployments and 100 entities
	b.Run("10000 flows and 100 entities to be pruned", benchmarkRemoveOrphanedFlows(flowStore, eStore, fixtureconsts.Deployment1, 100, 100))
	// 100 deployments and 1000 entities
	b.Run("100000 flows and 1000 entities to be pruned", benchmarkRemoveOrphanedFlows(flowStore, eStore, fixtureconsts.Deployment1, 100, 1000))

	// 1 deployments and 1000 entities
	b.Run("1000 flows and 1000 entities to be pruned", benchmarkRemoveOrphanedFlows(flowStore, eStore, fixtureconsts.Deployment1, 1, 1000))
	// 1 deployments and 10000 entities
	b.Run("10000 flows and 10000 entities to be pruned", benchmarkRemoveOrphanedFlows(flowStore, eStore, fixtureconsts.Deployment1, 1, 10000))
	// 1 deployments and 100000 entities
	b.Run("100000 flows and 100000 entities to be pruned", benchmarkRemoveOrphanedFlows(flowStore, eStore, fixtureconsts.Deployment1, 1, 100000))
}

func BenchmarkGetFlowsForDeployment(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := postgresFlowStore.NewClusterStore(psql)
	flowStore, err := clusterStore.CreateFlowStore(ctx, fixtureconsts.Cluster1)
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

	err := flowStore.UpsertFlows(ctx, flows, timestamp.Now()-1000000)
	require.NoError(b, err)
}

func setupExternalIngressFlows(b *testing.B, flowStore store.FlowStore, deploymentId string, numFlows uint32) {
	flows := make([]*storage.NetworkFlow, 0, numFlows)
	for i := uint32(0); i < numFlows; i++ {
		id, err := testutils.ExtIdFromIPv4(fixtureconsts.Cluster1, i)
		require.NoError(b, err)
		flows = append(flows, testutils.ExtFlowIngress(deploymentId, id.String(), clusterID))
	}

	err := flowStore.UpsertFlows(ctx, flows, timestamp.Now()-1000000)
	require.NoError(b, err)
}

func addToUUID(u string, addition int64) string {
	uBytes, _ := uuid.Parse(u)

	bi := new(big.Int)
	bi.SetBytes(uBytes[:])
	bi.Add(bi, big.NewInt(addition))

	newBytes := bi.Bytes()
	newUUID, _ := uuid.FromBytes(newBytes)

	return newUUID.String()
}

func upsertTooMany(b *testing.B, eStore entityStore.EntityDataStore, entities []*storage.NetworkEntity) {
	batchSize := 3000
	numEntities := len(entities)

	for offset := 0; offset < numEntities; offset += batchSize {
		end := offset + batchSize
		if end > numEntities {
			end = numEntities
		}
		_, err := eStore.CreateExtNetworkEntitiesForCluster(allAccessCtx, fixtureconsts.Cluster1, entities[offset:end]...)
		require.NoError(b, err)
	}
}

func setupExternalFlowsWithEntities(b *testing.B, flowStore store.FlowStore, eStore entityStore.EntityDataStore, startingDeploymentId string, numDeployments int, numEntities uint32, ts timestamp.MicroTS, startingIPIndex uint32) {
	totalFlows := numEntities * uint32(numDeployments)
	flows := make([]*storage.NetworkFlow, 0, totalFlows)
	entities := make([]*storage.NetworkEntity, numEntities)

	for i := uint32(0); i < numEntities; i++ {
		bs := [4]byte{}
		// Must have + 1 because the 0.0.0.0 IP address is not allowed
		binary.BigEndian.PutUint32(bs[:], startingIPIndex+i+1)
		ip := netip.AddrFrom4(bs)
		cidr := fmt.Sprintf("%s/32", ip.String())

		entities[i] = GetClusterScopedDiscoveredEntity(cidr, clusterID)
	}

	upsertTooMany(b, eStore, entities)

	deploymentId := startingDeploymentId
	var flow *storage.NetworkFlow
	for i := 0; i < numDeployments; i++ {
		for j := uint32(0); j < numEntities; j++ {
			id := entities[j].GetInfo().GetId()
			if j%2 == 0 {
				flow = testutils.ExtFlowEgress(id, deploymentId, clusterID)
			} else {
				flow = testutils.ExtFlowIngress(deploymentId, id, clusterID)
			}
			flows = append(flows, flow)
		}
		deploymentId = addToUUID(deploymentId, 1)
	}

	err := flowStore.UpsertFlows(ctx, flows, ts)
	require.NoError(b, err)
}

func setupExternalEgressFlows(b *testing.B, flowStore store.FlowStore, deploymentId string, numFlows uint32) {
	flows := make([]*storage.NetworkFlow, 0, numFlows)
	for i := uint32(0); i < numFlows; i++ {
		id, err := testutils.ExtIdFromIPv4(fixtureconsts.Cluster1, i)
		require.NoError(b, err)

		flows = append(flows, testutils.ExtFlowEgress(id.String(), deploymentId, clusterID))
	}

	err := flowStore.UpsertFlows(ctx, flows, timestamp.Now()-1000000)
	require.NoError(b, err)
}

func benchmarkGetExternalFlows(flowStore store.FlowStore, deploymentId string) func(*testing.B) {
	return func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := flowStore.GetExternalFlowsForDeployment(ctx, deploymentId)
			require.NoError(b, err)
		}
	}
}

func benchmarkGetFlowsForDeployment(flowStore store.FlowStore, deploymentId string) func(*testing.B) {
	return func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := flowStore.GetFlowsForDeployment(ctx, deploymentId)
			require.NoError(b, err)
		}
	}
}

func benchmarkPruneOrphanedFlowsForDeployment(flowStore store.FlowStore, eStore entityStore.EntityDataStore, deploymentId string, numEntities uint32) func(*testing.B) {
	return func(b *testing.B) {
		// Prune all flows and entities from previous tests
		orphanWindow := time.Now().UTC().Add(20000000 * time.Second)
		err := flowStore.RemoveOrphanedFlows(allAccessCtx, &orphanWindow)

		ts := timestamp.Now() + 1000000
		// Add flows and entities that will be pruned
		startingIPIndex := uint32(0)
		setupExternalFlowsWithEntities(b, flowStore, eStore, deploymentId, 1, numEntities, ts, startingIPIndex)

		// Add flows and entities that will not be pruned
		startingIPIndex = uint32(numEntities)
		deploymentId = addToUUID(deploymentId, 1)
		setupExternalFlowsWithEntities(b, flowStore, eStore, deploymentId, 1, numEntities, ts, startingIPIndex)
		start := time.Now()
		b.ResetTimer()

		b.Setenv(features.ExternalIPs.EnvVar(), "true")
		err = flowStore.RemoveFlowsForDeployment(allAccessCtx, deploymentId)
		require.NoError(b, err)
		duration := time.Since(start)

		log.Infof("Pruning %d flows and entities took %s", numEntities, duration)
	}
}

func benchmarkRemoveOrphanedFlows(flowStore store.FlowStore, eStore entityStore.EntityDataStore, deploymentId string, numDeployments int, numEntities uint32) func(*testing.B) {
	totalFlows := uint32(numDeployments) * numEntities
	return func(b *testing.B) {
		// Prune all flows and entities from previous tests
		orphanWindow := time.Now().UTC().Add(20000000 * time.Second)
		err := flowStore.RemoveOrphanedFlows(allAccessCtx, &orphanWindow)

		// Add flows and entities that will be pruned
		ts := timestamp.Now() - 10000000
		startingIPIndex := uint32(0)
		setupExternalFlowsWithEntities(b, flowStore, eStore, deploymentId, numDeployments, numEntities, ts, startingIPIndex)

		// Add flows and entities that will not be pruned
		deploymentId = addToUUID(deploymentId, int64(numDeployments))
		ts = timestamp.Now() + 10000000
		startingIPIndex = uint32(numEntities)
		setupExternalFlowsWithEntities(b, flowStore, eStore, deploymentId, numDeployments, numEntities, ts, startingIPIndex)

		start := time.Now()
		b.ResetTimer()

		b.Setenv(features.ExternalIPs.EnvVar(), "true")
		orphanWindow = time.Now().UTC()
		err = flowStore.RemoveOrphanedFlows(allAccessCtx, &orphanWindow)
		require.NoError(b, err)
		duration := time.Since(start)
		log.Infof("Pruning %d flows and %d entities took %s", totalFlows, numEntities, duration)
	}
}

func exclude(b *testing.B, f func()) {
	b.StopTimer()
	f()
	b.StartTimer()
}
