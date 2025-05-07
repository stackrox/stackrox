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
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	ngTestutils "github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

	now := time.Now().UTC()
	timeNow := timestamp.FromGoTime(now)

	time1 := timeNow.Add(time.Second)

	networkFlow := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "a"},
			DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: "b"},
			DstPort:    1,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		LastSeenTimestamp: time1.Protobuf(),
		ClusterId:         clusterID,
	}

	foundNetworkFlows, _, err := s.flowStore.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	// Adding the same thing twice to ensure that we only retrieve 1 based on serial Flow_Id implementation
	s.NoError(s.flowStore.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, timeNow))
	networkFlow.LastSeenTimestamp = timeNow.Add(2 * time.Second).Protobuf()
	s.NoError(s.flowStore.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, timeNow))
	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 1)
	assert.True(s.T(), testhelper.MatchElements([]*storage.NetworkFlow{networkFlow}, foundNetworkFlows))

	// Check the get all flows by since time
	time3 := now.Add(3 * time.Second)
	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, &time3)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	s.NoError(s.flowStore.RemoveFlow(s.ctx, networkFlow.GetProps()))
	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	s.NoError(s.flowStore.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, timeNow))

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

	s.NoError(s.flowStore.UpsertFlows(s.ctx, networkFlows, timeNow))

	foundNetworkFlows, _, err = s.flowStore.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, flowCount)

	// Make sure store for second cluster does not find any flows
	foundNetworkFlows, _, err = store2.GetAllFlows(s.ctx, nil)
	s.NoError(err)
	s.Len(foundNetworkFlows, 0)

	// Add a flow to the second cluster
	networkFlow.ClusterId = secondCluster
	s.NoError(store2.UpsertFlows(s.ctx, []*storage.NetworkFlow{networkFlow}, timeNow))

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

// Create a discovered External-Source entity that is ClusterScoped
func GetClusterScopedDiscoveredEntity(ip string, clusterID string) *storage.NetworkEntity {
	id, err := externalsrcs.NewClusterScopedID(clusterID, ip)
	if err != nil {
		return nil
	}
	return ngTestutils.GetExtSrcNetworkEntity(id.String(), ip, ip, false, clusterID, true)
}

type flowsWithTimestamp struct {
	flows     []*storage.NetworkFlow
	updatedAt timestamp.MicroTS
}

func getNFlows(flows []flowsWithTimestamp) int {
	count := 0
	for _, flow := range flows {
		count += len(flow.flows)
	}

	return count
}

// Two flows using two distinct external-entities. Both are pruned
// and we expect that all entities are pruned as well.
func (s *NetworkflowStoreSuite) TestPruneExternalEntitiesAllOrphaned() {
	s.T().Setenv(features.ExternalIPs.EnvVar(), "true")

	now := time.Now()

	extEntity1 := GetClusterScopedDiscoveredEntity("223.42.0.1/32", clusterID)
	extEntity2 := GetClusterScopedDiscoveredEntity("223.42.0.2/32", clusterID)

	egressFlow := &storage.NetworkFlow{
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
		ClusterId: clusterID,
	}

	ingressFlow1 := &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			DstPort: 22,
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   extEntity1.GetInfo().Id,
			},
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   fixtureconsts.Deployment1,
			},
		},
		ClusterId: clusterID,
	}

	ingressFlow2 := &storage.NetworkFlow{
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
		ClusterId: clusterID,
	}

	cases := []struct {
		name             string
		flows            []flowsWithTimestamp
		entities         []*storage.NetworkEntity
		expectedFlows    int
		expectedEntities int
		window           time.Time
	}{
		{
			name: "Prune everything due to timestamp",
			flows: []flowsWithTimestamp{
				{
					flows: []*storage.NetworkFlow{
						egressFlow,
						ingressFlow2,
					},
					updatedAt: timestamp.FromGoTime(now.Add(-100 * time.Second)),
				},
			},
			entities: []*storage.NetworkEntity{
				extEntity1,
				extEntity2,
			},
			expectedFlows:    0,
			expectedEntities: 0,
			window:           now.UTC().Add(-10 * time.Second),
		},
		{
			// One entity used by two flows. One flow pruned only.
			// We expect that the entity remains.
			name: "Do not prune entity due to one flow still using it",
			flows: []flowsWithTimestamp{
				{
					flows: []*storage.NetworkFlow{
						egressFlow,
					},
					updatedAt: timestamp.FromGoTime(now.Add(-300 * time.Second)),
				},
				{
					flows: []*storage.NetworkFlow{
						ingressFlow1,
					},
					updatedAt: timestamp.FromGoTime(now.Add(-100 * time.Second)),
				},
			},
			entities: []*storage.NetworkEntity{
				extEntity1,
			},
			expectedFlows:    1,
			expectedEntities: 1,
			window:           now.UTC().Add(-200 * time.Second),
		},
		{
			// Two flows using two distinct external-entities. Neither are pruned
			// and we expect that all entities are still there at the end.
			name: "Nothing is pruned",
			flows: []flowsWithTimestamp{
				{
					flows: []*storage.NetworkFlow{
						egressFlow,
						ingressFlow2,
					},
					updatedAt: timestamp.FromGoTime(now.Add(-10 * time.Second)),
				},
			},
			entities: []*storage.NetworkEntity{
				extEntity1,
				extEntity2,
			},
			expectedFlows:    2,
			expectedEntities: 2,
			window:           now.UTC().Add(-100 * time.Second),
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			s.SetupTest()

			for _, entity := range c.entities {
				err := s.entityStore.UpdateExternalNetworkEntity(s.ctx, entity, false)
				s.Nil(err)
			}

			for _, flows := range c.flows {
				err := s.flowStore.UpsertFlows(s.ctx, flows.flows, flows.updatedAt)
				s.Nil(err)
			}

			nflows := getNFlows(c.flows)

			// flows initially in the DB
			row := s.pgDB.DB.QueryRow(s.ctx, flowsCountStmt)
			var count int
			err := row.Scan(&count)
			s.Nil(err)
			s.Equal(nflows, count)

			// entities initially in the DB
			row = s.pgDB.DB.QueryRow(s.ctx, entitiesCountStmt)
			err = row.Scan(&count)
			s.Nil(err)
			s.Equal(len(c.entities), count)

			// pruning (anything older than 10s).
			// Flows should get pruned because Deployment1 is not in the DB.
			err = s.flowStore.RemoveOrphanedFlows(s.ctx, &c.window)
			s.Nil(err)

			// flows after pruning
			row = s.pgDB.DB.QueryRow(s.ctx, flowsCountStmt)
			err = row.Scan(&count)
			s.Nil(err)
			s.Equal(c.expectedFlows, count)

			// entities after pruning
			row = s.pgDB.DB.QueryRow(s.ctx, entitiesCountStmt)
			err = row.Scan(&count)
			s.Nil(err)
			s.Equal(c.expectedEntities, count)
		})
	}
}

// Three flows: Ext1->Depl1 Ext1->Depl2 Depl2->Ext2
// Remove Depl2
// Expect 1 flow and 1 entity remaining
func (s *NetworkflowStoreSuite) TestRemoveDeplExternalEntitiesOrphaned() {
	s.T().Setenv(features.ExternalIPs.EnvVar(), "true")

	now := time.Now()

	extEntity1 := GetClusterScopedDiscoveredEntity("223.42.0.1/32", clusterID)
	err := s.entityStore.UpdateExternalNetworkEntity(s.ctx, extEntity1, false)
	s.Nil(err)

	extEntity2 := GetClusterScopedDiscoveredEntity("223.42.0.2/32", clusterID)
	err = s.entityStore.UpdateExternalNetworkEntity(s.ctx, extEntity2, false)
	s.Nil(err)

	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					Id:   extEntity1.GetInfo().Id,
				},
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   fixtureconsts.Deployment1,
				},
			},
			ClusterId: clusterID,
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					Id:   extEntity1.GetInfo().Id,
				},
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   fixtureconsts.Deployment2,
				},
			},
			ClusterId: clusterID,
		},
		{
			Props: &storage.NetworkFlowProperties{
				DstPort: 22,
				SrcEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   fixtureconsts.Deployment2,
				},
				DstEntity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
					Id:   extEntity2.GetInfo().Id,
				},
			},
			ClusterId: clusterID,
		},
	}

	err = s.flowStore.UpsertFlows(s.ctx, flows, timestamp.FromGoTime(now.Add(-100*time.Second)))
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

	// Delete deployment2
	err = s.flowStore.RemoveFlowsForDeployment(s.ctx, fixtureconsts.Deployment2)
	s.Nil(err)

	// flows after pruning
	row = s.pgDB.DB.QueryRow(s.ctx, flowsCountStmt)
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(1, count)

	// entities after pruning
	row = s.pgDB.DB.QueryRow(s.ctx, entitiesCountStmt)
	err = row.Scan(&count)
	s.Nil(err)
	s.Equal(1, count)
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
