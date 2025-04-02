//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	configDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	networkFlowDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	"github.com/stackrox/rox/central/networkgraph/testhelper"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	testCluster   = fixtureconsts.Cluster1
	testNamespace = fixtureconsts.Namespace1
)

func TestNetworkGraphService(t *testing.T) {
	suite.Run(t, new(networkGraphServiceSuite))
}

type networkGraphServiceSuite struct {
	suite.Suite

	db *pgtest.TestPostgres

	service Service

	clusterDataStore     clusterDS.DataStore
	deploymentsDataStore deploymentDS.DataStore
	entityDataStore      networkEntityDS.EntityDataStore
	flowDataStore        networkFlowDS.ClusterDataStore
	policyDataStore      networkPolicyDS.DataStore
	configDataStore      configDS.DataStore
	treeMgr              networktree.Manager
}

// using SetupTest/TeardownTest instead of SetupSuite/TeardownSuite
// to ensure new, blank, DB for each Test function
func (s *networkGraphServiceSuite) SetupTest() {
	db := pgtest.ForT(s.T())

	var err error
	s.clusterDataStore, err = clusterDS.GetTestPostgresDataStore(s.T(), db.DB)
	s.NoError(err)

	s.deploymentsDataStore, err = deploymentDS.GetTestPostgresDataStore(s.T(), db.DB)
	s.NoError(err)

	s.entityDataStore = networkEntityDS.GetTestPostgresDataStore(s.T(), db.DB)

	s.flowDataStore, err = networkFlowDS.GetTestPostgresClusterDataStore(s.T(), db.DB)
	s.NoError(err)

	s.policyDataStore, err = networkPolicyDS.GetTestPostgresDataStore(s.T(), db.DB)
	s.NoError(err)

	s.configDataStore = configDS.GetTestPostgresDataStore(s.T(), db.DB)

	s.treeMgr = networktree.Singleton()

	s.service = New(
		s.flowDataStore,
		s.entityDataStore,
		s.treeMgr,
		s.deploymentsDataStore,
		s.clusterDataStore,
		s.policyDataStore,
		s.configDataStore,
	)

	s.db = db
}

func externalFlow(deployment *storage.Deployment, entity *storage.NetworkEntity, ingress bool) *storage.NetworkFlow {
	deploymentEntityInfo := &storage.NetworkEntityInfo{
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		Id:   deployment.Id,
		Desc: &storage.NetworkEntityInfo_Deployment_{
			Deployment: &storage.NetworkEntityInfo_Deployment{
				Namespace: deployment.Namespace,
			},
		},
	}

	entityInfo := entity.GetInfo()

	if ingress {
		return &storage.NetworkFlow{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  entityInfo,
				DstEntity:  deploymentEntityInfo,
				DstPort:    1234,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: nil,
			ClusterId:         deployment.ClusterId,
		}
	}
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			DstEntity:  entityInfo,
			SrcEntity:  deploymentEntityInfo,
			DstPort:    1234,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		LastSeenTimestamp: nil,
		ClusterId:         deployment.ClusterId,
	}
}

func (s *networkGraphServiceSuite) TestGetExternalNetworkFlows() {
	ctx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.NetworkGraph),
		),
	)

	globalWriteAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph, resources.Deployment)))

	entityID1, _ := externalsrcs.NewClusterScopedID(testCluster, "192.168.1.1/32")
	entityID2, _ := externalsrcs.NewClusterScopedID(testCluster, "10.0.0.2/32")
	entityID3, _ := externalsrcs.NewClusterScopedID(testCluster, "1.1.1.1/32")

	entityIDs := []string{
		entityID1.String(), entityID2.String(), entityID3.String(),
	}

	entities := []*storage.NetworkEntity{
		testutils.GetExtSrcNetworkEntity(entityIDs[0], "ext1", "192.168.1.1/32", false, testCluster),
		testutils.GetExtSrcNetworkEntity(entityIDs[1], "ext2", "10.0.0.2/32", false, testCluster),
		testutils.GetExtSrcNetworkEntity(entityIDs[2], "ext3", "10.0.100.25/32", false, testCluster),
	}

	deployments := []*storage.Deployment{
		{
			Id:        fixtureconsts.Deployment1,
			ClusterId: testCluster,
			Namespace: testNamespace,
		},

		{
			Id:        fixtureconsts.Deployment2,
			ClusterId: testCluster,
			Namespace: testNamespace,
		},
	}

	for _, deployment := range deployments {
		err := s.deploymentsDataStore.UpsertDeployment(globalWriteAccessCtx, deployment)
		s.NoError(err)
	}

	for _, e := range entities {
		err := s.entityDataStore.CreateExternalNetworkEntity(globalWriteAccessCtx, e, true)
		s.NoError(err)
	}

	singleEntityFlow := externalFlow(deployments[0], entities[0], false)
	multiEntityFlows := []*storage.NetworkFlow{
		externalFlow(deployments[0], entities[2], false),
		externalFlow(deployments[1], entities[2], false),
	}

	allFlows := []*storage.NetworkFlow{singleEntityFlow}
	allFlows = append(allFlows, multiEntityFlows...)

	flowStore, err := s.flowDataStore.CreateFlowStore(globalWriteAccessCtx, testCluster)
	s.NoError(err)

	err = flowStore.UpsertFlows(globalWriteAccessCtx, allFlows, timestamp.FromGoTime(time.Now()))
	s.NoError(err)

	for _, tc := range []struct {
		name          string
		request       *v1.GetExternalNetworkFlowsRequest
		expected      *v1.GetExternalNetworkFlowsResponse
		expectSuccess bool
	}{
		{
			name: "Get single entity flows",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: testCluster,
				EntityId:  entityIDs[0],
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: &v1.GetExternalNetworkFlowsResponse{
				Entity: entities[0].GetInfo(),
				Flows: []*storage.NetworkFlow{
					singleEntityFlow,
				},
			},
			expectSuccess: true,
		},
		{
			name: "Entity with no flows",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: testCluster,
				EntityId:  entityIDs[1],
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: &v1.GetExternalNetworkFlowsResponse{
				Entity: entities[1].GetInfo(),
				Flows:  []*storage.NetworkFlow{},
			},
			expectSuccess: true,
		},
		{
			name: "Invalid entity ID",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: testCluster,
				EntityId:  "invalid ID",
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected:      nil,
			expectSuccess: false,
		},
		{
			name: "Invalid cluster",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: "invalid cluster",
				EntityId:  entityIDs[0],
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected:      nil,
			expectSuccess: false,
		},
		{
			name: "entity with multiple flows",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: testCluster,
				EntityId:  entityIDs[2],
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: &v1.GetExternalNetworkFlowsResponse{
				Entity: entities[2].GetInfo(),
				Flows:  multiEntityFlows,
			},
			expectSuccess: true,
		},
	} {
		s.Run(tc.name, func() {
			response, err := s.service.GetExternalNetworkFlows(ctx, tc.request)
			if tc.expectSuccess {
				s.NoError(err)
				protoassert.Equal(s.T(), tc.expected.Entity, response.Entity)
				assert.True(s.T(), testhelper.MatchElements(tc.expected.Flows, response.Flows))
			} else {
				s.Error(err)
			}
		})
	}
}

func (s *networkGraphServiceSuite) TestGetExternalNetworkFlowsMetadata() {
	ctx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.NetworkGraph),
		),
	)

	globalWriteAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph, resources.Deployment)))

	entityID, _ := externalsrcs.NewClusterScopedID(testCluster, "192.168.1.1/32")
	entityID2, _ := externalsrcs.NewClusterScopedID(testCluster, "10.0.0.2/32")
	entityID3, _ := externalsrcs.NewClusterScopedID(testCluster, "1.1.1.1/32")

	entityIDs := []string{
		entityID.String(), entityID2.String(), entityID3.String(),
	}

	entities := []*storage.NetworkEntity{
		testutils.GetExtSrcNetworkEntity(entityIDs[0], "ext1", "192.168.1.1/32", false, testCluster),
		testutils.GetExtSrcNetworkEntity(entityIDs[1], "ext2", "10.0.0.2/32", false, testCluster),
		testutils.GetExtSrcNetworkEntity(entityIDs[2], "ext3", "10.0.100.25/32", false, testCluster),
	}

	deployments := []*storage.Deployment{
		{
			Id:        fixtureconsts.Deployment1,
			ClusterId: testCluster,
			Namespace: testNamespace,
		},

		{
			Id:        fixtureconsts.Deployment2,
			ClusterId: testCluster,
			Namespace: testNamespace,
		},

		{
			Id:        fixtureconsts.Deployment3,
			ClusterId: testCluster,
			Namespace: fixtureconsts.Namespace2,
		},
	}

	for _, deployment := range deployments {
		err := s.deploymentsDataStore.UpsertDeployment(globalWriteAccessCtx, deployment)
		s.NoError(err)
	}

	for _, e := range entities {
		err := s.entityDataStore.CreateExternalNetworkEntity(globalWriteAccessCtx, e, true)
		s.NoError(err)
	}

	flows := []*storage.NetworkFlow{
		// deployment1 -> 192.168.1.1
		externalFlow(deployments[0], entities[0], false),
		externalFlow(deployments[0], entities[2], false),

		// deployment2 -> (10.0.0.2, 10.0.100.25)
		externalFlow(deployments[1], entities[2], false),
		externalFlow(deployments[1], entities[1], false),

		// different namespace to 10.0.100.25
		externalFlow(deployments[2], entities[2], false),
	}

	flowStore, err := s.flowDataStore.CreateFlowStore(globalWriteAccessCtx, testCluster)
	s.NoError(err)

	err = flowStore.UpsertFlows(globalWriteAccessCtx, flows, timestamp.FromGoTime(time.Now()))
	s.NoError(err)

	for _, tc := range []struct {
		name          string
		request       *v1.GetExternalNetworkFlowsMetadataRequest
		expected      *v1.GetExternalNetworkFlowsMetadataResponse
		expectSuccess bool
	}{
		{
			name: "All entities",
			request: &v1.GetExternalNetworkFlowsMetadataRequest{
				ClusterId: testCluster,
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: &v1.GetExternalNetworkFlowsMetadataResponse{
				Entities: []*v1.ExternalNetworkFlowMetadata{
					{
						Entity:     entities[0].GetInfo(),
						FlowsCount: 1,
					},
					{
						Entity:     entities[1].GetInfo(),
						FlowsCount: 1,
					},
					{
						Entity:     entities[2].GetInfo(),
						FlowsCount: 2,
					},
				},
				TotalEntities: 3,
			},
			expectSuccess: true,
		},
		{
			name: "Filter CIDR with wide subnet",
			request: &v1.GetExternalNetworkFlowsMetadataRequest{
				ClusterId: testCluster,
				Query:     fmt.Sprintf("Namespace:%s+External Source Address:10.0.0.0/8", testNamespace),
			},
			expected: &v1.GetExternalNetworkFlowsMetadataResponse{
				Entities: []*v1.ExternalNetworkFlowMetadata{
					{
						Entity:     entities[1].GetInfo(),
						FlowsCount: 1,
					},
					{
						Entity:     entities[2].GetInfo(),
						FlowsCount: 2,
					},
				},
				TotalEntities: 2,
			},
			expectSuccess: true,
		},
		{
			name: "Filter CIDR with narrow subnet",
			request: &v1.GetExternalNetworkFlowsMetadataRequest{
				ClusterId: testCluster,
				Query:     fmt.Sprintf("Namespace:%s+External Source Address:10.0.0.0/24", testNamespace),
			},
			expected: &v1.GetExternalNetworkFlowsMetadataResponse{
				Entities: []*v1.ExternalNetworkFlowMetadata{
					{
						Entity:     entities[1].GetInfo(),
						FlowsCount: 1,
					},
				},
				TotalEntities: 1,
			},
			expectSuccess: true,
		},
		{
			name: "Get metadata from different namespace",
			request: &v1.GetExternalNetworkFlowsMetadataRequest{
				ClusterId: testCluster,
				Query:     fmt.Sprintf("Namespace:%s", fixtureconsts.Namespace2),
			},
			expected: &v1.GetExternalNetworkFlowsMetadataResponse{
				Entities: []*v1.ExternalNetworkFlowMetadata{
					{
						Entity:     entities[2].GetInfo(),
						FlowsCount: 1,
					},
				},
				TotalEntities: 1,
			},
			expectSuccess: true,
		},
		{
			name: "Invalid cluster",
			request: &v1.GetExternalNetworkFlowsMetadataRequest{
				ClusterId: "invalid cluster",
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected:      nil,
			expectSuccess: false,
		},
		{
			name: "Invalid namespace",
			request: &v1.GetExternalNetworkFlowsMetadataRequest{
				ClusterId: testCluster,
				Query:     "Namespace:invalidNamespace",
			},
			expected: &v1.GetExternalNetworkFlowsMetadataResponse{
				Entities:      []*v1.ExternalNetworkFlowMetadata{},
				TotalEntities: 0,
			},
			expectSuccess: true,
		},
		{
			name: "Paginate the response",
			request: &v1.GetExternalNetworkFlowsMetadataRequest{
				ClusterId: testCluster,
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
				Pagination: &v1.Pagination{
					Offset: 0,
					Limit:  1,
				},
			},
			expected: &v1.GetExternalNetworkFlowsMetadataResponse{
				Entities: []*v1.ExternalNetworkFlowMetadata{
					{
						Entity:     entities[0].GetInfo(),
						FlowsCount: 1,
					},
				},
				TotalEntities: 3,
			},
			expectSuccess: true,
		},
	} {
		s.Run(tc.name, func() {
			response, err := s.service.GetExternalNetworkFlowsMetadata(ctx, tc.request)
			if tc.expectSuccess {
				s.NoError(err)

				s.Assert().Len(response.Entities, len(tc.expected.Entities))
				s.Assert().Equal(response.TotalEntities, tc.expected.TotalEntities)
			} else {
				s.Error(err)
			}
		})
	}
}
