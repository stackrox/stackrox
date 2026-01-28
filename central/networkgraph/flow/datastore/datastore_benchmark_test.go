//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/deployment/cache"
	"github.com/stackrox/rox/central/networkgraph/aggregator"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	postgresFlowStore "github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/require"
)

// BenchmarkDatastoreUpsertFlows benchmarks the UpsertFlows operation at the datastore layer,
// including SAC checks and filtering of deleted deployments.
func BenchmarkDatastoreUpsertFlows(b *testing.B) {
	psql := pgtest.ForT(b)

	clusterStore := postgresFlowStore.NewClusterStore(psql)
	underlyingFlowStore, err := clusterStore.CreateFlowStore(context.Background(), fixtureconsts.Cluster1)
	require.NoError(b, err)

	// Create a mock datastore with all required dependencies
	datastoreImpl := &flowDataStoreImpl{
		storage:                   underlyingFlowStore,
		graphConfig:               newMockGraphConfigDS(),
		hideDefaultExtSrcsManager: newMockNetworkConnsAggregator(),
		deletedDeploymentsCache:   newMockDeletedDeployments(),
	}

	// These benchmarks test the datastore layer with different flow batch sizes.
	// Note: Results should be compared with consideration for the total number of flows upserted,
	// as the datastore layer adds SAC checks and deployment filtering on top of the store layer.
	b.Run("datastore upsert single flow", benchmarkDatastoreUpsertFlows(datastoreImpl, 1))
	b.Run("datastore upsert 100 flow batch", benchmarkDatastoreUpsertFlows(datastoreImpl, 100))
	b.Run("datastore upsert 1000 flow batch", benchmarkDatastoreUpsertFlows(datastoreImpl, 1000))
	b.Run("datastore upsert 10000 flow batch", benchmarkDatastoreUpsertFlows(datastoreImpl, 10000))
	b.Run("datastore upsert 50000 flow batch", benchmarkDatastoreUpsertFlows(datastoreImpl, 50000))
	b.Run("benchmark upsert 100000 flow batch", benchmarkDatastoreUpsertFlows(datastoreImpl, 100000))
}

func benchmarkDatastoreUpsertFlows(ds *flowDataStoreImpl, numFlows uint32) func(*testing.B) {
	return func(b *testing.B) {
		flows := make([]*storage.NetworkFlow, 0, numFlows)
		for i := uint32(0); i < numFlows; i++ {
			id, err := testutils.ExtIdFromIPv4(fixtureconsts.Cluster1, i)
			require.NoError(b, err)
			flows = append(flows, testutils.ExtFlowIngress(fixtureconsts.Deployment1, id.String(), fixtureconsts.Cluster1))
		}

		// Create a context with appropriate SAC permissions for the benchmark
		ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes())

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := ds.UpsertFlows(ctx, flows, timestamp.Now()-1000000)
			require.NoError(b, err)

			exclude(b, func() {
				err = ds.RemoveFlowsForDeployment(ctx, fixtureconsts.Deployment1)
				require.NoError(b, err)
			})
		}
	}
}

// Helper functions for mock dependencies

func newMockGraphConfigDS() graphConfigDS.DataStore {
	// Return a mock that returns a default config (no hiding of external sources)
	return &mockGraphConfigDS{}
}

func newMockNetworkConnsAggregator() aggregator.NetworkConnsAggregator {
	// Return a pass-through aggregator that doesn't modify flows
	return &mockNetworkConnsAggregator{}
}

func newMockDeletedDeployments() cache.DeletedDeployments {
	return &mockDeletedDeployments{}
}

type mockGraphConfigDS struct{}

func (m *mockGraphConfigDS) GetNetworkGraphConfig(ctx context.Context) (*storage.NetworkGraphConfig, error) {
	return &storage.NetworkGraphConfig{
		HideDefaultExternalSrcs: false,
	}, nil
}

func (m *mockGraphConfigDS) UpdateNetworkGraphConfig(ctx context.Context, config *storage.NetworkGraphConfig) error {
	return nil
}

type mockNetworkConnsAggregator struct{}

func (m *mockNetworkConnsAggregator) Aggregate(flows []*storage.NetworkFlow) []*storage.NetworkFlow {
	// Pass through: don't filter any flows
	return flows
}

type mockDeletedDeployments struct{}

func (m *mockDeletedDeployments) Add(deployment string) {
	// No-op
}

func (m *mockDeletedDeployments) Contains(deployment string) bool {
	// No-op
	return false
}

func exclude(b *testing.B, f func()) {
	b.StopTimer()
	f()
	b.StartTimer()
}
