//go:build sql_integration

package manager

import (
	"context"
	"testing"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nbDS "github.com/stackrox/rox/central/networkbaseline/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	networkFlowDSMocks "github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func generateBaselines(b *testing.B) []*storage.NetworkBaseline {
	var networkBaselines []*storage.NetworkBaseline
	for i := 0; i < 2000; i++ {
		networkBaseline := &storage.NetworkBaseline{}
		require.NoError(b, testutils.FullInit(networkBaseline, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		networkBaseline.Peers = nil
		networkBaseline.ForbiddenPeers = nil
		networkBaselines = append(networkBaselines, networkBaseline)
	}

	return networkBaselines
}

func BenchmarkInitFromStore(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	ctx := sac.WithAllAccess(context.Background())

	pgtestbase := pgtest.ForT(b)
	require.NotNil(b, pgtestbase)

	nbStore, err := nbDS.GetBenchPostgresDataStore(b, pgtestbase.DB)
	require.NoError(b, err)
	npStore, err := npDS.GetBenchPostgresDataStore(b, pgtestbase.DB)
	require.NoError(b, err)
	networkEntityStore, err := networkEntityDS.GetBenchPostgresDataStore(b, pgtestbase.DB)
	require.NoError(b, err)

	deploymentDS := deploymentMocks.NewMockDataStore(mockCtrl)
	clusterFlows := networkFlowDSMocks.NewMockClusterDataStore(mockCtrl)

	sensorCnxMgr := connectionMocks.NewMockManager(mockCtrl)

	// load it up
	require.NoError(b, nbStore.UpsertNetworkBaselines(ctx, generateBaselines(b)))

	for i := 0; i < 2000; i++ {
		networkPolicy := &storage.NetworkPolicy{}
		require.NoError(b, testutils.FullInit(networkPolicy, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		require.NoError(b, npStore.UpsertNetworkPolicy(ctx, networkPolicy))
	}

	b.Run("New", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := New(nbStore, networkEntityStore, deploymentDS, npStore, clusterFlows, sensorCnxMgr)
			require.NoError(b, err)
		}
	})

	log.Infof("Welcome to benching.")
}
