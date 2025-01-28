//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	configDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	networkFlowDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
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

	s.entityDataStore, err = networkEntityDS.GetTestPostgresDataStore(s.T(), db.DB)
	s.NoError(err)

	s.flowDataStore, err = networkFlowDS.GetTestPostgresClusterDataStore(s.T(), db.DB)
	s.NoError(err)

	s.policyDataStore, err = networkPolicyDS.GetTestPostgresDataStore(s.T(), db.DB)
	s.NoError(err)

	s.configDataStore, err = configDS.GetTestPostgresDataStore(s.T(), db.DB)
	s.NoError(err)

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

func (s *networkGraphServiceSuite) TeardownTest() {
	s.db.Teardown(s.T())
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

	entityID, _ := externalsrcs.NewClusterScopedID(testCluster, "192.168.1.1/32")
	entityID2, _ := externalsrcs.NewClusterScopedID(testCluster, "10.0.0.2/32")
	entityID3, _ := externalsrcs.NewClusterScopedID(testCluster, "1.1.1.1/32")

	entity := testutils.GetExtSrcNetworkEntity(entityID.String(), "ext1", "192.168.1.1/32", false, testCluster)
	entity2 := testutils.GetExtSrcNetworkEntity(entityID2.String(), "ext2", "10.0.0.2/32", false, testCluster)
	entity3 := testutils.GetExtSrcNetworkEntity(entityID3.String(), "ext3", "10.0.100.25/32", false, testCluster)

	entities := []*storage.NetworkEntity{
		entity,
		entity2,
		entity3,
	}

	deployment := &storage.Deployment{
		Id:        fixtureconsts.Deployment1,
		ClusterId: testCluster,
		Namespace: testNamespace,
	}

	deployment2 := &storage.Deployment{
		Id:        fixtureconsts.Deployment2,
		ClusterId: testCluster,
		Namespace: testNamespace,
	}

	err := s.deploymentsDataStore.UpsertDeployment(globalWriteAccessCtx, deployment)
	s.NoError(err)

	err = s.deploymentsDataStore.UpsertDeployment(globalWriteAccessCtx, deployment2)
	s.NoError(err)

	for _, e := range entities {
		err = s.entityDataStore.CreateExternalNetworkEntity(globalWriteAccessCtx, e, true)
		s.NoError(err)
	}

	deploymentToEntity1 := externalFlow(deployment, entity, false)
	deploymentToEntity3 := externalFlow(deployment, entity3, false)

	deployment2ToEntity3 := externalFlow(deployment2, entity3, false)

	// flow from deployment to es1a and es1c but not es1b
	flows := []*storage.NetworkFlow{
		deploymentToEntity1, deploymentToEntity3,
		deployment2ToEntity3,
	}

	flowStore, err := s.flowDataStore.CreateFlowStore(globalWriteAccessCtx, testCluster)
	s.NoError(err)

	err = flowStore.UpsertFlows(globalWriteAccessCtx, flows, timestamp.FromGoTime(time.Now()))
	s.NoError(err)

	since := time.Now().Add(-1 * time.Hour)
	flows, _, err = flowStore.GetAllFlows(ctx, &since)

	for _, tc := range []struct {
		name     string
		request  *v1.GetExternalNetworkFlowsRequest
		expected *v1.GetExternalNetworkFlowsResponse
		pass     bool
	}{
		{
			name: "Get single entity flows",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: testCluster,
				EntityId:  entityID.String(),
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: &v1.GetExternalNetworkFlowsResponse{
				Entity: entity.GetInfo(),
				Flows: []*storage.NetworkFlow{
					deploymentToEntity1,
				},
			},
			pass: true,
		},
		{
			name: "Entity with no flows",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: testCluster,
				EntityId:  entityID2.String(),
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: &v1.GetExternalNetworkFlowsResponse{
				Entity: entity2.GetInfo(),
				Flows:  []*storage.NetworkFlow{},
			},
			pass: true,
		},
		{
			name: "Invalid entity ID",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: testCluster,
				EntityId:  "invalid ID",
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: nil,
			pass:     false,
		},
		{
			name: "Invalid cluster",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: "invalid cluster",
				EntityId:  entityID.String(),
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: nil,
			pass:     false,
		},
		{
			name: "entity with multiple flows",
			request: &v1.GetExternalNetworkFlowsRequest{
				ClusterId: testCluster,
				EntityId:  entityID3.String(),
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: &v1.GetExternalNetworkFlowsResponse{
				Entity: entity3.GetInfo(),
				Flows: []*storage.NetworkFlow{
					deploymentToEntity3,
					deployment2ToEntity3,
				},
			},
			pass: true,
		},
	} {
		s.Run(tc.name, func() {
			response, err := s.service.GetExternalNetworkFlows(ctx, tc.request)
			if tc.pass {
				s.NoError(err)
				protoassert.Equal(s.T(), tc.expected.Entity, response.Entity)
				protoassert.ElementsMatch(s.T(), tc.expected.Flows, response.Flows)
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

	entity := testutils.GetExtSrcNetworkEntity(entityID.String(), "ext1", "192.168.1.1/32", false, testCluster)
	entity2 := testutils.GetExtSrcNetworkEntity(entityID2.String(), "ext2", "10.0.0.2/32", false, testCluster)
	entity3 := testutils.GetExtSrcNetworkEntity(entityID3.String(), "ext3", "10.0.100.25/32", false, testCluster)

	entities := []*storage.NetworkEntity{
		entity,
		entity2,
		entity3,
	}

	deployment := &storage.Deployment{
		Id:        fixtureconsts.Deployment1,
		ClusterId: testCluster,
		Namespace: testNamespace,
	}

	deployment2 := &storage.Deployment{
		Id:        fixtureconsts.Deployment2,
		ClusterId: testCluster,
		Namespace: testNamespace,
	}

	deploymentDifferentNamespace := &storage.Deployment{
		Id:        fixtureconsts.Deployment3,
		ClusterId: testCluster,
		Namespace: fixtureconsts.Namespace2,
	}

	err := s.deploymentsDataStore.UpsertDeployment(globalWriteAccessCtx, deployment)
	s.NoError(err)

	err = s.deploymentsDataStore.UpsertDeployment(globalWriteAccessCtx, deployment2)
	s.NoError(err)

	err = s.deploymentsDataStore.UpsertDeployment(globalWriteAccessCtx, deploymentDifferentNamespace)
	s.NoError(err)

	for _, e := range entities {
		err = s.entityDataStore.CreateExternalNetworkEntity(globalWriteAccessCtx, e, true)
		s.NoError(err)
	}

	deploymentToEntity1 := externalFlow(deployment, entity, false)
	deploymentToEntity3 := externalFlow(deployment, entity3, false)

	deployment2ToEntity3 := externalFlow(deployment2, entity3, false)
	deployment2ToEntity2 := externalFlow(deployment2, entity2, false)

	deploymentDiffNSToEntity3 := externalFlow(deploymentDifferentNamespace, entity3, false)

	// flow from deployment to es1a and es1c but not es1b
	flows := []*storage.NetworkFlow{
		deploymentToEntity1, deploymentToEntity3,
		deployment2ToEntity2, deployment2ToEntity3,
		deploymentDiffNSToEntity3,
	}

	flowStore, err := s.flowDataStore.CreateFlowStore(globalWriteAccessCtx, testCluster)
	s.NoError(err)

	err = flowStore.UpsertFlows(globalWriteAccessCtx, flows, timestamp.FromGoTime(time.Now()))
	s.NoError(err)

	since := time.Now().Add(-1 * time.Hour)
	flows, _, err = flowStore.GetAllFlows(ctx, &since)

	for _, tc := range []struct {
		name     string
		request  *v1.GetExternalNetworkFlowsMetadataRequest
		expected *v1.GetExternalNetworkFlowsMetadataResponse
		pass     bool
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
						Entity:     entity.GetInfo(),
						FlowsCount: 1,
					},
					{
						Entity:     entity2.GetInfo(),
						FlowsCount: 1,
					},
					{
						Entity:     entity3.GetInfo(),
						FlowsCount: 2,
					},
				},
				TotalEntities: 3,
			},
			pass: true,
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
						Entity:     entity2.GetInfo(),
						FlowsCount: 1,
					},
					{
						Entity:     entity3.GetInfo(),
						FlowsCount: 2,
					},
				},
				TotalEntities: 2,
			},
			pass: true,
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
						Entity:     entity2.GetInfo(),
						FlowsCount: 1,
					},
				},
				TotalEntities: 1,
			},
			pass: true,
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
						Entity:     entity3.GetInfo(),
						FlowsCount: 1,
					},
				},
				TotalEntities: 1,
			},
			pass: true,
		},
		{
			name: "Invalid cluster",
			request: &v1.GetExternalNetworkFlowsMetadataRequest{
				ClusterId: "invalid cluster",
				Query:     fmt.Sprintf("Namespace:%s", testNamespace),
			},
			expected: nil,
			pass:     false,
		},
		{
			name: "Invalid namespace",
			request: &v1.GetExternalNetworkFlowsMetadataRequest{
				ClusterId: testCluster,
				Query:     fmt.Sprintf("Namespace:invalidNamespace"),
			},
			expected: &v1.GetExternalNetworkFlowsMetadataResponse{
				Entities:      []*v1.ExternalNetworkFlowMetadata{},
				TotalEntities: 0,
			},
			pass: true,
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
						Entity:     entity.GetInfo(),
						FlowsCount: 1,
					},
				},
				TotalEntities: 3,
			},
			pass: true,
		},
	} {
		s.Run(tc.name, func() {
			response, err := s.service.GetExternalNetworkFlowsMetadata(ctx, tc.request)
			if tc.pass {
				s.NoError(err)

				if tc.request.Pagination != nil {
					// if paginated response, just verify length, since the
					// elements themselves are not deterministic
					s.Assert().Len(response.Entities, len(tc.expected.Entities))
					s.Assert().Equal(response.TotalEntities, tc.expected.TotalEntities)
				} else {
					protoassert.ElementsMatch(s.T(), tc.expected.Entities, response.Entities)
					s.Assert().Equal(response.TotalEntities, tc.expected.TotalEntities)
				}
			} else {
				s.Error(err)
			}
		})
	}
}
