package postgres

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
)

type NetworkflowStoreSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestNetworkflowStore(t *testing.T) {
	suite.Run(t, new(NetworkflowStoreSuite))
}

func (s *NetworkflowStoreSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	} else {
		s.envIsolator.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")
	}
}

func (s *NetworkflowStoreSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func getTimestamp(seconds int64) *types.Timestamp {
	return &types.Timestamp{
		Seconds: seconds,
	}
}

func (s *NetworkflowStoreSuite) TestStore() {
	ctx := context.Background()
	clusterID := "22"
	secondCluster := "43"

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.NoError(err)
	defer pool.Close()

	Destroy(ctx, pool)
	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	store := CreateTableAndNewStore(ctx, pool, gormDB, clusterID)
	store2 := CreateTableAndNewStore(ctx, pool, gormDB, secondCluster)

	networkFlow := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "a"},
			DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "b"},
			DstPort:    1,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		LastSeenTimestamp: getTimestamp(1),
		ClusterId:         "22",
	}
	zeroTs := timestamp.MicroTS(0)

	foundNetworkFlows, _, err := store.GetAllFlows(ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	// Adding the same thing twice to ensure that we only retrieve 1 based on serial Flow_Id implementation
	s.NoError(store.UpsertFlows(ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))
	networkFlow.LastSeenTimestamp = getTimestamp(2)
	s.NoError(store.UpsertFlows(ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))
	foundNetworkFlows, _, err = store.GetAllFlows(ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 1)
	s.Equal(networkFlow, foundNetworkFlows[0])

	// Check the get all flows by since time
	foundNetworkFlows, _, err = store.GetAllFlows(ctx, getTimestamp(3))
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	s.NoError(store.RemoveFlow(ctx, networkFlow.GetProps()))
	foundNetworkFlows, _, err = store.GetAllFlows(ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	s.NoError(store.UpsertFlows(ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))

	err = store.RemoveFlowsForDeployment(ctx, networkFlow.GetProps().GetSrcEntity().GetId())
	s.NoError(err)

	foundNetworkFlows, _, err = store.GetAllFlows(ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	var networkFlows []*storage.NetworkFlow
	flowCount := 100
	for i := 0; i < flowCount; i++ {
		networkFlow := &storage.NetworkFlow{}
		s.NoError(testutils.FullInit(networkFlow, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		networkFlows = append(networkFlows, networkFlow)
	}

	s.NoError(store.UpsertFlows(ctx, networkFlows, zeroTs))

	foundNetworkFlows, _, err = store.GetAllFlows(ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, flowCount)

	// Make sure store for second cluster does not find any flows
	foundNetworkFlows, _, err = store2.GetAllFlows(ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	// Add a flow to the second cluster
	networkFlow.ClusterId = secondCluster
	s.NoError(store2.UpsertFlows(ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))

	foundNetworkFlows, _, err = store2.GetAllFlows(ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 1)

	// Clean up
	Destroy(ctx, pool)
}
