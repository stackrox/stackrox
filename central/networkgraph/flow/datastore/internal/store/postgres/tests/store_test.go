//go:build sql_integration

package tests

import (
	"context"
	"testing"
	"time"

	entityStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	postgresFlowStore "github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkgraph/testhelper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	ngTestutils "github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"

	"gorm.io/gorm"
)

const (
	clusterID = fixtureconsts.Cluster1

	flowsCountStmt    = "select count(*) from network_flows_v2"
	entitiesCountStmt = "select count(*) from network_entities"
)

type NetworkflowStoreSuite struct {
	suite.Suite
	flowStore   postgresFlowStore.FlowStore
	entityStore entityStore.EntityDataStore
	ctx         context.Context
	pgDB        *pgtest.TestPostgres
	gormDB      *gorm.DB
}

func TestNetworkflowStore(t *testing.T) {
	suite.Run(t, new(NetworkflowStoreSuite))
}

func (s *NetworkflowStoreSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
}

func (s *NetworkflowStoreSuite) SetupTest() {
	s.pgDB = pgtest.ForT(s.T())

	s.flowStore = postgresFlowStore.CreateTableAndNewStore(s.ctx, s.pgDB.DB, s.gormDB, clusterID)
	s.entityStore = entityStore.GetTestPostgresDataStore(s.T(), s.pgDB.DB)
	s.entityStore.RegisterCluster(s.ctx, clusterID)
}

func (s *NetworkflowStoreSuite) TearDownTest() {
	err := s.entityStore.DeleteExternalNetworkEntitiesForCluster(s.ctx, clusterID)
	s.Nil(err)
}

func (s *NetworkflowStoreSuite) TearDownSuite() {
	if s.gormDB != nil {
		pgtest.CloseGormDB(s.T(), s.gormDB)
	}
}

func (s *NetworkflowStoreSuite) TestStore() {
	secondCluster := fixtureconsts.Cluster2
	store2 := postgresFlowStore.New(s.pgDB.DB, secondCluster)

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

	foundNetworkFlows, _, err := s.flowStore.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	// Adding the same thing twice to ensure that we only retrieve 1 based on serial Flow_Id implementation
	s.NoError(s.flowStore.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))
	networkFlow.LastSeenTimestamp = protocompat.GetProtoTimestampFromSeconds(2)
	s.NoError(s.flowStore.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))
	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 1)
	assert.True(s.T(), testhelper.MatchElements([]*storage.NetworkFlow{networkFlow}, foundNetworkFlows))

	// Check the get all flows by since time
	time3 := time.Unix(3, 0)
	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, &time3)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	s.NoError(s.flowStore.RemoveFlow(s.ctx, networkFlow.GetProps()))
	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	s.NoError(s.flowStore.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, zeroTs))

	err = s.flowStore.RemoveFlowsForDeployment(s.ctx, networkFlow.GetProps().GetSrcEntity().GetId())
	s.NoError(err)

	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	var networkFlows []*storage.NetworkFlow
	flowCount := 100
	for i := 0; i < flowCount; i++ {
		networkFlow := &storage.NetworkFlow{}
		s.NoError(testutils.FullInit(networkFlow, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		networkFlows = append(networkFlows, networkFlow)
	}

	s.NoError(s.flowStore.UpsertFlows(s.ctx, networkFlows, zeroTs))

	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, nil)
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
	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, nil)
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

	err := s.flowStore.UpsertFlows(s.ctx, flows, timestamp.Now())
	s.Nil(err)

	row := s.pgDB.DB.QueryRow(s.ctx, flowsCountStmt)
	var count int
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(count, len(flows))

	err = s.flowStore.RemoveStaleFlows(s.ctx)
	s.Nil(err)

	row = s.pgDB.DB.QueryRow(s.ctx, flowsCountStmt)
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(count, 2)
}

func (s *NetworkflowStoreSuite) TestPruneOrphanedExternalEntities() {
	s.T().Setenv(features.ExternalIPs.EnvVar(), "true")

	now := time.Now()

	extEntity1 := ngTestutils.GetDiscoveredExtSrcNetworkEntity("223.42.0.1/32", clusterID)
	err := s.entityStore.UpdateExternalNetworkEntity(s.ctx, extEntity1, false)
	s.Nil(err)

	extEntity2 := ngTestutils.GetDiscoveredExtSrcNetworkEntity("223.42.0.2/32", clusterID)
	err = s.entityStore.UpdateExternalNetworkEntity(s.ctx, extEntity2, false)
	s.Nil(err)

	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   fixtureconsts.Deployment1,
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					Id:   extEntity1.GetInfo().Id,
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: timestamppb.New(now.Add(-1000)),
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					Id:   extEntity2.GetInfo().Id,
				},
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   fixtureconsts.Deployment1,
				},
			},
			ClusterId:         clusterID,
			LastSeenTimestamp: timestamppb.New(now.Add(-1000)),
		},
	}

	err = s.flowStore.UpsertFlows(s.ctx, flows, timestamp.FromGoTime(now))
	s.Nil(err)

	// flows initially in the DB
	row := s.pgDB.DB.QueryRow(s.ctx, flowsCountStmt)
	var count int
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(len(flows), count)

	// entities initially in the DB
	row = s.pgDB.DB.QueryRow(s.ctx, entitiesCountStmt)
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(2, count)

	// pruning
	window := now.Add(-100)
	err = s.flowStore.RemoveOrphanedFlows(s.ctx, &window)
	s.Nil(err)

	// flows after pruning
	row = s.pgDB.DB.QueryRow(s.ctx, flowsCountStmt)
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(0, count)

	// entities after pruning
	row = s.pgDB.DB.QueryRow(s.ctx, entitiesCountStmt)
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(0, count)
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

	err = s.flowStore.UpsertFlows(s.ctx, flows, timestamp.Now())
	s.Nil(err)

	// Normalize flow timestamps

	filteredFlows, _, err := s.flowStore.GetMatchingFlows(s.ctx, deploymentIngressFlowsPredicate, nil)
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

	err = s.flowStore.UpsertFlows(s.ctx, flows, timestamp.Now())
	s.Nil(err)

	deploymentFlows, err := s.flowStore.GetFlowsForDeployment(s.ctx, "TestDeploymentSrc1")
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

	err = s.flowStore.UpsertFlows(s.ctx, flows, timestamp.Now())
	s.Nil(err)

	deploymentFlows, err := s.flowStore.GetExternalFlowsForDeployment(s.ctx, "TestDeployment1")
	s.Nil(err)
	assert.True(s.T(), testhelper.MatchElements([]*storage.NetworkFlow{flows[0], flows[1]}, deploymentFlows))
}
