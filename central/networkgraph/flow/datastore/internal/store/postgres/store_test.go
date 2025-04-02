//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/networkgraph/testhelper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

const (
	clusterID = fixtureconsts.Cluster1

	flowsCountStmt = "select count(*) from network_flows_v2"
)

type NetworkflowStoreSuite struct {
	suite.Suite
	store  FlowStore
	ctx    context.Context
	pool   postgres.DB
	gormDB *gorm.DB
}

func TestNetworkflowStore(t *testing.T) {
	suite.Run(t, new(NetworkflowStoreSuite))
}

func (s *NetworkflowStoreSuite) SetupSuite() {
	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	s.pool, err = postgres.New(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
}

func (s *NetworkflowStoreSuite) SetupTest() {
	Destroy(s.ctx, s.pool)
	s.store = CreateTableAndNewStore(s.ctx, s.pool, s.gormDB, clusterID)
}

func (s *NetworkflowStoreSuite) TearDownTest() {
	if s.pool != nil {
		// Clean up
		Destroy(s.ctx, s.pool)
	}
}

func (s *NetworkflowStoreSuite) TearDownSuite() {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.gormDB != nil {
		pgtest.CloseGormDB(s.T(), s.gormDB)
	}
}

func (s *NetworkflowStoreSuite) TestStore() {
	secondCluster := fixtureconsts.Cluster2
	store2 := New(s.pool, secondCluster)

	networkFlow := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "a"},
			DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "b"},
			DstPort:    1,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		LastSeenTimestamp: protocompat.GetProtoTimestampFromSeconds(1),
		ClusterId:         clusterID,
	}
	zeroTs := timestamp.MicroTS(0)

	foundNetworkFlows, _, err := s.store.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	// Adding the same thing twice to ensure that we only retrieve 1 based on serial Flow_Id implementation
	s.NoError(s.store.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))
	networkFlow.LastSeenTimestamp = protocompat.GetProtoTimestampFromSeconds(2)
	s.NoError(s.store.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))
	foundNetworkFlows, _, err = s.store.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 1)
	assert.True(s.T(), testhelper.MatchElements([]*storage.NetworkFlow{networkFlow}, foundNetworkFlows))

	// Check the get all flows by since time
	time3 := time.Unix(3, 0)
	foundNetworkFlows, _, err = s.store.GetAllFlows(s.ctx, &time3)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	s.NoError(s.store.RemoveFlow(s.ctx, networkFlow.GetProps()))
	foundNetworkFlows, _, err = s.store.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	s.NoError(s.store.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))

	err = s.store.RemoveFlowsForDeployment(s.ctx, networkFlow.GetProps().GetSrcEntity().GetId())
	s.NoError(err)

	foundNetworkFlows, _, err = s.store.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	var networkFlows []*storage.NetworkFlow
	flowCount := 100
	for i := 0; i < flowCount; i++ {
		networkFlow := &storage.NetworkFlow{}
		s.NoError(testutils.FullInit(networkFlow, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		networkFlows = append(networkFlows, networkFlow)
	}

	s.NoError(s.store.UpsertFlows(s.ctx, networkFlows, zeroTs))

	foundNetworkFlows, _, err = s.store.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, flowCount)

	// Make sure store for second cluster does not find any flows
	foundNetworkFlows, _, err = store2.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	// Add a flow to the second cluster
	networkFlow.ClusterId = secondCluster
	s.NoError(store2.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))

	foundNetworkFlows, _, err = store2.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 1)

	pred := func(props *storage.NetworkFlowProperties) bool {
		return true
	}
	foundNetworkFlows, _, err = store2.GetMatchingFlows(s.ctx, pred, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 1)

	// Store 1 flows should remain
	foundNetworkFlows, _, err = s.store.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, flowCount)
}

func (s *NetworkflowStoreSuite) TestPruneStaleNetworkFlows() {
	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestDst1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: nil,
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestDst2",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestSrc2",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: protocompat.TimestampNow(),
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestDst1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: protocompat.TimestampNow(),
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestDst1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: protocompat.TimestampNow(),
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestDst1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: protocompat.TimestampNow(),
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestDst2",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: 2,
					Id:   "TestSrc2",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: nil,
		},
	}

	err := s.store.UpsertFlows(s.ctx, flows, timestamp.Now())
	s.Nil(err)

	row := s.pool.QueryRow(s.ctx, flowsCountStmt)
	var count int
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(count, len(flows))

	err = s.store.RemoveStaleFlows(s.ctx)
	s.Nil(err)

	row = s.pool.QueryRow(s.ctx, flowsCountStmt)
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(count, 2)
}

func deploymentIngressFlowsPredicate(props *storage.NetworkFlowProperties) bool {
	return props.GetDstEntity().GetType() == storage.NetworkEntityInfo_DEPLOYMENT
}

func (s *NetworkflowStoreSuite) TestGetMatching() {
	now, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Truncate(time.Microsecond))
	s.Require().NoError(err)

	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   "TestInternetDst1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeploymentSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: nil,
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeploymentDst1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   "TestInternetSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: now,
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeploymentDst2",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeploymentSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: now,
		},
	}

	err = s.store.UpsertFlows(s.ctx, flows, timestamp.Now())
	s.Nil(err)

	// Normalize flow timestamps

	filteredFlows, _, err := s.store.GetMatchingFlows(s.ctx, deploymentIngressFlowsPredicate, nil)
	s.Nil(err)
	assert.True(s.T(), testhelper.MatchElements([]*storage.NetworkFlow{flows[1], flows[2]}, filteredFlows))
}

func (s *NetworkflowStoreSuite) TestGetFlowsForDeployment() {
	now, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Truncate(time.Microsecond))
	s.Require().NoError(err)

	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   "TestInternetDst1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeploymentSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: nil,
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeploymentDst1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   "TestInternetSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: now,
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeploymentDst2",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeploymentSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: now,
		},
	}

	err = s.store.UpsertFlows(s.ctx, flows, timestamp.Now())
	s.Nil(err)

	deploymentFlows, err := s.store.GetFlowsForDeployment(s.ctx, "TestDeploymentSrc1")
	s.Nil(err)
	assert.True(s.T(), testhelper.MatchElements([]*storage.NetworkFlow{flows[0], flows[2]}, deploymentFlows))
}

func (s *NetworkflowStoreSuite) TestGetExternalFlowsForDeployment() {
	now, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Truncate(time.Microsecond))
	s.Require().NoError(err)

	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					Id:   "TestExternalDst1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeployment1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: nil,
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeployment1",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					Id:   "TestExternalSrc1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: now,
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeploymentDst2",
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   "TestDeployment1",
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: now,
		},
	}

	err = s.store.UpsertFlows(s.ctx, flows, timestamp.Now())
	s.Nil(err)

	deploymentFlows, err := s.store.GetExternalFlowsForDeployment(s.ctx, "TestDeployment1")
	s.Nil(err)
	assert.True(s.T(), testhelper.MatchElements([]*storage.NetworkFlow{flows[0], flows[1]}, deploymentFlows))
}
